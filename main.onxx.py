from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from contextlib import asynccontextmanager

from PIL import Image
from transformers import ViTImageProcessor
import onnxruntime as ort
import numpy as np

import io
import os
import time
import logging
from typing import Dict, Any

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
# App lifespan (load model once)
# -------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):
    global session, processor

    logger.info("Loading ONNX model...")
    start = time.time()

    if not os.path.exists(MODEL_PATH):
        raise RuntimeError(f"ONNX model not found at {MODEL_PATH}")

    session = ort.InferenceSession(
        MODEL_PATH,
        providers=["CPUExecutionProvider"]
    )

    processor = ViTImageProcessor.from_pretrained(MODEL_DIR)

    logger.info(f"ONNX model loaded in {time.time() - start:.2f}s")
    yield
    logger.info("Shutting down...")

# -------------------------------------------------
# FastAPI app
# -------------------------------------------------
app = FastAPI(
    title="NSFW Image Detection API (ONNX)",
    description="Fast & lightweight NSFW detection using ONNX Runtime",
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
# Inference logic
# -------------------------------------------------
def process_image(image_bytes: bytes) -> Dict[str, Any]:
    if session is None or processor is None:
        raise RuntimeError("Model not loaded")

    img = Image.open(io.BytesIO(image_bytes)).convert("RGB")

    inputs = processor(images=img, return_tensors="np")

    ort_inputs = {
        session.get_inputs()[0].name: inputs["pixel_values"]
    }

    logits = session.run(None, ort_inputs)[0]

    # Softmax
    exp = np.exp(logits - logits.max(axis=-1, keepdims=True))
    probs = exp / exp.sum(axis=-1, keepdims=True)

    idx = int(probs.argmax(axis=-1)[0])
    confidence = float(probs[0][idx])

    # ⚠️ Labels — verify once if needed
    labels = {0: "safe", 1: "nsfw"}

    return {
        "label": labels[idx],
        "confidence": round(confidence, 4),
        "scores": {
            labels[0]: round(float(probs[0][0]), 4),
            labels[1]: round(float(probs[0][1]), 4),
        }
    }

# -------------------------------------------------
# Routes
# -------------------------------------------------
@app.get("/")
async def root():
    return {
        "status": "online",
        "runtime": "onnxruntime",
        "model_path": MODEL_PATH,
        "version": "3.0.0"
    }

@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "model_loaded": session is not None,
        "provider": session.get_providers() if session else None
    }

@app.post("/detect")
async def detect_nsfw(file: UploadFile = File(...)):
    start = time.time()

    try:
        contents = await file.read()
        result = process_image(contents)
        result["processing_time_ms"] = round((time.time() - start) * 1000, 2)
        return JSONResponse(content=result)

    except Exception as e:
        logger.exception("Detection failed")
        raise HTTPException(status_code=500, detail=str(e))