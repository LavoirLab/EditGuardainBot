import asyncio
import io
import os
import time
import logging
import tempfile
from contextlib import asynccontextmanager
from typing import List, Dict, Any, Optional

import cv2
import torch
from PIL import Image
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from transformers import AutoModelForImageClassification, ViTImageProcessor
import uvicorn

MODEL_NAME = "Falconsai/nsfw_image_detection"
BATCH_SIZE = 16
BATCH_TIMEOUT = 0.04
MAX_FILE_SIZE_MB = 50
MAX_BATCH_FILES = 20

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("nsfw_api")

model = None
processor = None
device = None
request_queue: asyncio.Queue = asyncio.Queue()


class InferenceRequest:
    def __init__(self, images: List[Image.Image]):
        self.images = images
        self.future: asyncio.Future = asyncio.get_event_loop().create_future()


@asynccontextmanager
async def lifespan(app: FastAPI):
    global model, processor, device

    logger.info("Loading NSFW detection model...")
    start = time.time()

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    logger.info(f"Using device: {device}")

    model = AutoModelForImageClassification.from_pretrained(
        MODEL_NAME,
        torch_dtype=torch.float16 if device.type == "cuda" else torch.float32,
    ).to(device)
    model.eval()

    processor = ViTImageProcessor.from_pretrained(MODEL_NAME)

    logger.info(f"Model loaded in {time.time() - start:.2f}s on {device}")

    worker_task = asyncio.create_task(batch_worker())

    yield

    worker_task.cancel()
    logger.info("Shutting down...")
    del model
    del processor
    if torch.cuda.is_available():
        torch.cuda.empty_cache()


app = FastAPI(
    title="NSFW Media Detection API",
    description="NSFW content detection for images, videos, GIFs, and stickers",
    version="3.0.0",
    lifespan=lifespan,
)

app.add_middleware(GZipMiddleware, minimum_size=1000)
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


async def batch_worker():
    while True:
        batch: List[InferenceRequest] = []
        deadline = time.monotonic() + BATCH_TIMEOUT

        while len(batch) < BATCH_SIZE:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                break
            try:
                req = await asyncio.wait_for(request_queue.get(), timeout=remaining)
                batch.append(req)
            except asyncio.TimeoutError:
                break

        if not batch:
            await asyncio.sleep(0.001)
            continue

        all_images: List[Image.Image] = []
        mapping: List[int] = []
        for i, req in enumerate(batch):
            for img in req.images:
                all_images.append(img)
                mapping.append(i)

        try:
            raw_results = _run_inference(all_images)
        except Exception as e:
            logger.error(f"Batch inference error: {e}")
            for req in batch:
                if not req.future.done():
                    req.future.set_exception(e)
            continue

        per_request: List[List[Dict]] = [[] for _ in batch]
        for idx, result in enumerate(raw_results):
            per_request[mapping[idx]].append(result)

        for req, frame_results in zip(batch, per_request):
            if not req.future.done():
                req.future.set_result(_aggregate_frames(frame_results))


def _run_inference(images: List[Image.Image]) -> List[Dict[str, Any]]:
    if not images:
        return []

    inputs = processor(images=images, return_tensors="pt")
    inputs = {k: v.to(device) for k, v in inputs.items()}

    with torch.no_grad():
        logits = model(**inputs).logits
        probs = torch.nn.functional.softmax(logits, dim=-1)

    results = []
    for i in range(len(images)):
        pred_idx = logits[i].argmax(-1).item()
        label = model.config.id2label[pred_idx]
        confidence = probs[i][pred_idx].item()
        scores = {
            model.config.id2label[j]: round(probs[i][j].item(), 4)
            for j in range(len(model.config.id2label))
        }
        results.append({
            "label": label,
            "confidence": round(confidence, 4),
            "scores": scores,
        })
    return results


def _aggregate_frames(frame_results: List[Dict[str, Any]]) -> Dict[str, Any]:
    if not frame_results:
        return {"label": "error", "confidence": 0.0, "scores": {}}

    nsfw = [r for r in frame_results if r["label"].lower() == "nsfw"]
    best = max(nsfw if nsfw else frame_results, key=lambda x: x["confidence"])
    best["frames_analyzed"] = len(frame_results)
    return best


def extract_frames(contents: bytes, filename: str) -> List[Image.Image]:
    name = (filename or "").lower()
    ext = os.path.splitext(name)[1]

    if ext in (".mp4", ".webm", ".mov", ".avi", ".mkv"):
        return _extract_video_frames(contents, ext)

    if ext in (".gif", ".webp"):
        try:
            img = Image.open(io.BytesIO(contents))
            if getattr(img, "is_animated", False):
                n = img.n_frames
                indices = sorted(set([0, n // 3, (2 * n) // 3, max(n - 1, 0)]))
                frames = []
                for i in indices:
                    try:
                        img.seek(i)
                        frames.append(img.convert("RGB"))
                    except EOFError:
                        break
                return frames
            else:
                return [img.convert("RGB")]
        except Exception as e:
            logger.error(f"Failed to open animated image {filename}: {e}")

    try:
        return [Image.open(io.BytesIO(contents)).convert("RGB")]
    except Exception as e:
        logger.error(f"Failed to open image {filename}: {e}")
        return []


def _extract_video_frames(contents: bytes, suffix: str) -> List[Image.Image]:
    tmp_path: Optional[str] = None
    try:
        with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
            tmp.write(contents)
            tmp_path = tmp.name

        cap = cv2.VideoCapture(tmp_path)
        if not cap.isOpened():
            return []

        total = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        if total <= 0:
            cap.release()
            return []

        indices = sorted(set([0, total // 2, max(total - 1, 0)]))
        frames = []
        for idx in indices:
            cap.set(cv2.CAP_PROP_POS_FRAMES, idx)
            ok, frame = cap.read()
            if ok:
                frames.append(Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)))
        cap.release()
        return frames

    except Exception as e:
        logger.error(f"Error extracting video frames: {e}")
        return []
    finally:
        if tmp_path and os.path.exists(tmp_path):
            os.remove(tmp_path)


def _check_file_size(contents: bytes, filename: str):
    size_mb = len(contents) / (1024 * 1024)
    if size_mb > MAX_FILE_SIZE_MB:
        raise HTTPException(
            status_code=413,
            detail=f"File {filename!r} exceeds {MAX_FILE_SIZE_MB} MB limit ({size_mb:.1f} MB)",
        )


@app.get("/")
async def root():
    return {
        "status": "online",
        "model": MODEL_NAME,
        "device": str(device),
        "version": "3.0.0",
    }


@app.get("/health")
async def health_check():
    return {
        "status": "healthy",
        "model_loaded": model is not None,
        "device": str(device),
        "gpu_available": torch.cuda.is_available(),
        "gpu_name": torch.cuda.get_device_name(0) if torch.cuda.is_available() else None,
    }


@app.post("/detect")
async def detect_nsfw(file: UploadFile = File(...)):
    start = time.time()
    contents = await file.read()
    _check_file_size(contents, file.filename)

    frames = extract_frames(contents, file.filename or "upload")
    if not frames:
        raise HTTPException(status_code=400, detail="Could not extract frames from media.")

    req = InferenceRequest(frames)
    await request_queue.put(req)

    try:
        result = await asyncio.wait_for(req.future, timeout=30.0)
    except asyncio.TimeoutError:
        raise HTTPException(status_code=504, detail="Inference timed out.")

    result["processing_time_ms"] = round((time.time() - start) * 1000, 2)
    return JSONResponse(content=result)


@app.post("/detect/batch")
async def detect_nsfw_batch(files: List[UploadFile] = File(...)):
    if len(files) > MAX_BATCH_FILES:
        raise HTTPException(
            status_code=400,
            detail=f"Maximum {MAX_BATCH_FILES} files per batch.",
        )

    start = time.time()
    futures: List[asyncio.Future] = []
    filenames: List[str] = []

    for file in files:
        contents = await file.read()
        try:
            _check_file_size(contents, file.filename)
        except HTTPException as e:
            fut: asyncio.Future = asyncio.get_event_loop().create_future()
            fut.set_exception(e)
            futures.append(fut)
            filenames.append(file.filename or "unknown")
            continue

        frames = extract_frames(contents, file.filename or "upload")
        if not frames:
            fut = asyncio.get_event_loop().create_future()
            fut.set_result({"label": "error", "confidence": 0.0, "scores": {}, "frames_analyzed": 0})
            futures.append(fut)
        else:
            req = InferenceRequest(frames)
            await request_queue.put(req)
            futures.append(req.future)
        filenames.append(file.filename or "unknown")

    results = []
    for i, fut in enumerate(futures):
        try:
            res = await asyncio.wait_for(asyncio.shield(fut), timeout=60.0)
        except Exception as e:
            res = {"label": "error", "confidence": 0.0, "error": str(e), "frames_analyzed": 0}
        res["filename"] = filenames[i]
        res["image_index"] = i
        results.append(res)

    total_time = time.time() - start
    return JSONResponse(content={
        "results": results,
        "total_files": len(files),
        "total_processing_time_ms": round(total_time * 1000, 2),
        "avg_time_per_file_ms": round((total_time / max(len(files), 1)) * 1000, 2),
    })


if __name__ == "__main__":
    import multiprocessing
    workers = 1 if torch.cuda.is_available() else min(multiprocessing.cpu_count(), 4)
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8000")),
        workers=workers,
        log_level="info",
        access_log=True,
    )
