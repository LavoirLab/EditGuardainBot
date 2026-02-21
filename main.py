from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from contextlib import asynccontextmanager
from PIL import Image
import torch
from transformers import AutoModelForImageClassification, ViTImageProcessor
import io, os
import uvicorn
from typing import List, Dict, Any
import time
import logging


logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global model and processor variables
model = None
processor = None
device = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup, cleanup on shutdown"""
    global model, processor, device
    
    logger.info("Loading NSFW detection model...")
    start_time = time.time()
    
    try:
        # Determine device (GPU if available, otherwise CPU)
        device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        logger.info(f"Using device: {device}")
        
        # Load model and processor
        model_name = "Falconsai/nsfw_image_detection"
        
        model = AutoModelForImageClassification.from_pretrained(
            model_name,
            torch_dtype=torch.float16 if device.type == "cuda" else torch.float32
        ).to(device)
        
        # Set model to evaluation mode for faster inference
        model.eval()
        
        # Enable JIT compilation for faster inference (if supported)
        if device.type == "cuda":
            model = torch.jit.optimize_for_inference(torch.jit.script(model))
        
        processor = ViTImageProcessor.from_pretrained(model_name)
        
        load_time = time.time() - start_time
        logger.info(f"Model loaded successfully in {load_time:.2f} seconds")
        
    except Exception as e:
        logger.error(f"Error loading model: {e}")
        raise
    
    yield
    
    # Cleanup
    logger.info("Shutting down...")
    del model
    del processor
    torch.cuda.empty_cache() if torch.cuda.is_available() else None


# Initialize FastAPI app with lifespan
app = FastAPI(
    title="NSFW Image Detection API",
    description="Ultra-fast NSFW content detection using Vision Transformer",
    version="2.0.0",
    lifespan=lifespan
)

# Add middleware for compression
app.add_middleware(GZipMiddleware, minimum_size=1000)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


def process_image(image_bytes: bytes) -> Dict[str, Any]:
    """
    Process a single image and return classification results
    
    Args:
        image_bytes: Raw image bytes
        
    Returns:
        Dictionary with classification results and confidence scores
    """
    try:
        # Open image from bytes
        img = Image.open(io.BytesIO(image_bytes)).convert("RGB")
        
        # Preprocess image
        inputs = processor(images=img, return_tensors="pt")
        
        # Move inputs to same device as model
        inputs = {k: v.to(device) for k, v in inputs.items()}
        
        # Run inference with no gradient calculation
        with torch.no_grad():
            outputs = model(**inputs)
            logits = outputs.logits
            
            # Get probabilities
            probs = torch.nn.functional.softmax(logits, dim=-1)
            
            # Get predicted class
            predicted_idx = logits.argmax(-1).item()
            predicted_label = model.config.id2label[predicted_idx]
            confidence = probs[0][predicted_idx].item()
            
            # Get scores for both classes
            scores = {
                model.config.id2label[i]: probs[0][i].item() 
                for i in range(len(model.config.id2label))
            }
        
        return {
            "label": predicted_label,
            "confidence": round(confidence, 4),
            "scores": {k: round(v, 4) for k, v in scores.items()}
        }
        
    except Exception as e:
        logger.error(f"Error processing image: {e}")
        raise HTTPException(status_code=400, detail=f"Error processing image: {str(e)}")


@app.get("/")
async def root():
    """API health check endpoint"""
    return {
        "status": "online",
        "model": "Falconsai/nsfw_image_detection",
        "device": str(device),
        "version": "2.0.0"
    }


@app.get("/health")
async def health_check():
    """Detailed health check"""
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "device": str(device),
        "gpu_available": torch.cuda.is_available(),
        "gpu_name": torch.cuda.get_device_name(0) if torch.cuda.is_available() else None
    }


@app.post("/detect")
async def detect_nsfw(file: UploadFile = File(...)):
    """
    Detect NSFW content in a single image
    
    Args:
        file: Image file (JPEG, PNG, etc.)
        
    Returns:
        Classification results with label, confidence, and detailed scores
    """
    start_time = time.time()
    
    try:
        # Read image bytes
        contents = await file.read()
        
        # Process image
        result = process_image(contents)
        
        # Add processing time
        result["processing_time_ms"] = round((time.time() - start_time) * 1000, 2)
        
        return JSONResponse(content=result)
        
    except Exception as e:
        logger.error(f"Error in detect endpoint: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/detect/batch")
async def detect_nsfw_batch(files: List[UploadFile] = File(...)):
    """
    Detect NSFW content in multiple images (batch processing)
    
    Args:
        files: List of image files
        
    Returns:
        List of classification results for each image
    """
    start_time = time.time()
    
    if len(files) > 50:
        raise HTTPException(
            status_code=400, 
            detail="Maximum 50 images per batch. Use multiple requests for larger batches."
        )
    
    try:
        results = []
        
        # Process images in batch
        for idx, file in enumerate(files):
            contents = await file.read()
            result = process_image(contents)
            result["image_index"] = idx
            result["filename"] = file.filename
            results.append(result)
        
        total_time = time.time() - start_time
        
        return JSONResponse(content={
            "results": results,
            "total_images": len(files),
            "total_processing_time_ms": round(total_time * 1000, 2),
            "avg_time_per_image_ms": round((total_time / len(files)) * 1000, 2)
        })
        
    except Exception as e:
        logger.error(f"Error in batch detect endpoint: {e}")
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    # Run with multiple workers for production
    # For development, use: uvicorn main:app --reload
    # For production, use the command below or gunicorn with uvicorn workers
    import multiprocessing

    # Calculate workers: use CPU count for maximum throughput
    # On some systems, workers might need to be limited by RAM
    cpu_count = multiprocessing.cpu_count()
    
    uvicorn.run(
        "main:app",  # Use string import for multiple workers
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8000")),
        workers=cpu_count,
        log_level="info",
        access_log=True
    )