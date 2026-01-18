from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from contextlib import asynccontextmanager

from PIL import Image
from transformers import ViTImageProcessor
import onnxruntime as ort
import numpy as np

import io, os, time, logging
from typing import List, Dict, Any

# -------------------------------------------------
# Logging
# -------------------------------------------------
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# -------------------------------------------------
# Globals
# -------------------------------------------------
session: ort.InferenceSession | None = None
processor: ViTImageProcessor | None = None

MODEL_DIR = "onnx_model"
MODEL_PATH = os.path.join(MODEL_DIR, "model.onnx")

# -------------------------------------------------
# Lifespan: load ONNX once
# -------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):
    global session, processor

    logger.info("Loading NSFW ONNX model...")
    start_time = time.time()

    try:
        if not os.path.exists(MODEL_PATH):
            raise RuntimeError(f"ONNX model not found at {MODEL_PATH}")

        session = ort.InferenceSession(
            MODEL_PATH,
            providers=["CPUExecutionProvider"]
        )

        processor = ViTImageProcessor.from_pretrained(MODEL_DIR)

        logger.info(
            f"ONNX model loaded in {time.time() - start_time:.2f} seconds"
        )

    except Exception as e:
        logger.error(f"Error loading ONNX model: {e}")
        raise

    yield

    logger.info("Shutting down...")
    session = None
    processor = None

# -------------------------------------------------
# FastAPI app
# -------------------------------------------------
app = FastAPI(
    title="NSFW Image Detection API",
    description="Ultra-fast NSFW content detection using ONNX Runtime",
    version="3.0.0",
    lifespan=lifespan
)

app.add_middleware(GZipMiddleware, minimum_size=1000)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# -------------------------------------------------
# Inference logic (same output as PyTorch)
# -------------------------------------------------
def process_image(image_bytes: bytes) -> Dict[str, Any]:
    if session is None or processor is None:
        raise RuntimeError("Model not loaded")

    try:
        img = Image.open(io.BytesIO(image_bytes)).convert("RGB")

        inputs = processor(images=img, return_tensors="np")

        ort_inputs = {
            session.get_inputs()[0].name: inputs["pixel_values"]
        }

        logits = session.run(None, ort_inputs)[0]

        # softmax
        exp = np.exp(logits - logits.max(axis=-1, keepdims=True))
        probs = exp / exp.sum(axis=-1, keepdims=True)

        predicted_idx = int(probs.argmax(axis=-1)[0])
        confidence = float(probs[0][predicted_idx])

        # same labels as original model
        id2label = {0: "safe", 1: "nsfw"}

        scores = {
            id2label[i]: float(probs[0][i])
            for i in range(len(id2label))
        }

        return {
            "label": id2label[predicted_idx],
            "confidence": round(confidence, 4),
            "scores": {k: round(v, 4) for k, v in scores.items()},
        }

    except Exception as e:
        logger.error(f"Error processing image: {e}")
        raise HTTPException(status_code=400, detail=str(e))

# -------------------------------------------------
# Routes (UNCHANGED)
# -------------------------------------------------
@app.get("/")
async def root():
    return {
        "status": "online",
        "model": "Falconsai/nsfw_image_detection (onnx)",
        "runtime": "onnxruntime",
        "version": "3.0.0",
    }

@app.get("/health")
async def health_check():
    return {
        "status": "healthy",
        "model_loaded": session is not None,
        "provider": session.get_providers() if session else None,
    }

@app.post("/detect")
async def detect_nsfw(file: UploadFile = File(...)):
    start_time = time.time()

    try:
        contents = await file.read()
        result = process_image(contents)
        result["processing_time_ms"] = round(
            (time.time() - start_time) * 1000, 2
        )
        return JSONResponse(content=result)

    except Exception as e:
        logger.error(f"Error in detect endpoint: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/detect/batch")
async def detect_nsfw_batch(files: List[UploadFile] = File(...)):
    start_time = time.time()

    if len(files) > 50:
        raise HTTPException(
            status_code=400,
            detail="Maximum 50 images per batch. Use multiple requests."
        )

    try:
        results = []

        for idx, file in enumerate(files):
            contents = await file.read()
            result = process_image(contents)
            result["image_index"] = idx
            result["filename"] = file.filename
            results.append(result)

        total_time = time.time() - start_time

        return JSONResponse(
            content={
                "results": results,
                "total_images": len(files),
                "total_processing_time_ms": round(total_time * 1000, 2),
                "avg_time_per_image_ms": round(
                    (total_time / len(files)) * 1000, 2
                ),
            }
        )

    except Exception as e:
        logger.error(f"Error in batch detect endpoint: {e}")
        raise HTTPException(status_code=500, detail=str(e))

# -------------------------------------------------
# Run
# -------------------------------------------------
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        app,
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8000")),
        workers=1,
        log_level="info",
        access_log=True,
    )