import asyncio
import io
import os
import sys
import time
import logging
import tempfile
from contextlib import asynccontextmanager
from typing import List, Dict, Any, Optional

import uvloop
import cv2
import torch
from PIL import Image
from fastapi import FastAPI, File, Form, UploadFile, HTTPException
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

VALID_MEDIA_TYPES = {
    "photo",
    "sticker_static",
    "video",
}

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


def decode_by_media_type(contents: bytes, media_type: str) -> List[Image.Image]:
    """
    Decode media into PIL frames based on the exact type reported by Go.
    No guessing — Go has already resolved what this is from Telegram metadata.

    photo          → single JPEG/PNG image
    sticker_static → static WebP image
    sticker_video  → handled entirely in Go (ffmpeg extracts frame, sent here as 'photo')
    animation      → Telegram animations are MP4 — Go sends these as 'video'
    video          → MP4, extract up to 3 frames via cv2
    """
    if media_type == "photo" or media_type == "sticker_static":
        try:
            return [Image.open(io.BytesIO(contents)).convert("RGB")]
        except Exception as e:
            logger.error(f"Failed to decode {media_type}: {e}")
            return []

    if media_type == "video":
        return _extract_video_frames(contents, ".mp4")

    logger.warning(f"Unknown media_type: {media_type!r}")
    return []


def _extract_video_frames(contents: bytes, suffix: str) -> List[Image.Image]:
    tmp_path: Optional[str] = None
    try:
        with tempfile.NamedTemporaryFile(suffix=suffix, delete=False) as tmp:
            tmp.write(contents)
            tmp_path = tmp.name

        cap = cv2.VideoCapture(tmp_path)
        if not cap.isOpened():
            logger.error(f"cv2 could not open video file (suffix={suffix})")
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
async def detect_nsfw(
    media_type: str = Form(...),
    file: Optional[UploadFile] = File(default=None),
    file_path: Optional[str] = Form(default=None),
):
    if media_type not in VALID_MEDIA_TYPES:
        raise HTTPException(
            status_code=422,
            detail=f"Invalid media_type {media_type!r}. Must be one of: {sorted(VALID_MEDIA_TYPES)}",
        )

    start = time.time()

    if file_path is not None:
        if not os.path.isfile(file_path):
            raise HTTPException(status_code=400, detail=f"file_path does not exist: {file_path!r}")
        with open(file_path, "rb") as f:
            contents = f.read()
        fname = os.path.basename(file_path)
    elif file is not None:
        contents = await file.read()
        _check_file_size(contents, file.filename)
        fname = file.filename or "upload"
    else:
        raise HTTPException(status_code=422, detail="Either 'file' or 'file_path' must be provided.")

    frames = decode_by_media_type(contents, media_type)
    if not frames:
        raise HTTPException(status_code=400, detail=f"Could not decode media of type {media_type!r} from {fname!r}.")

    req = InferenceRequest(frames)
    await request_queue.put(req)

    try:
        result = await asyncio.wait_for(req.future, timeout=30.0)
    except asyncio.TimeoutError:
        raise HTTPException(status_code=504, detail="Inference timed out.")

    result["processing_time_ms"] = round((time.time() - start) * 1000, 2)
    result["media_type"] = media_type
    return JSONResponse(content=result)


@app.post("/detect/batch")
async def detect_nsfw_batch(
    media_types: List[str] = Form(...),
    files: Optional[List[UploadFile]] = File(default=None),
    file_paths: Optional[List[str]] = Form(default=None),
):
    localhost_mode = file_paths is not None and len(file_paths) > 0
    source_count = len(file_paths) if localhost_mode else (len(files) if files else 0)

    if source_count == 0:
        raise HTTPException(status_code=422, detail="Either 'files' or 'file_paths' must be provided.")
    if source_count > MAX_BATCH_FILES:
        raise HTTPException(status_code=400, detail=f"Maximum {MAX_BATCH_FILES} files per batch.")
    if source_count != len(media_types):
        raise HTTPException(
            status_code=422,
            detail=f"Source count ({source_count}) must match media_types count ({len(media_types)}).",
        )

    for mt in media_types:
        if mt not in VALID_MEDIA_TYPES:
            raise HTTPException(
                status_code=422,
                detail=f"Invalid media_type {mt!r}. Must be one of: {sorted(VALID_MEDIA_TYPES)}",
            )

    start = time.time()
    futures: List[asyncio.Future] = []
    filenames: List[str] = []

    if localhost_mode:
        for fp, mt in zip(file_paths, media_types):
            fname = os.path.basename(fp)
            if not os.path.isfile(fp):
                fut: asyncio.Future = asyncio.get_event_loop().create_future()
                fut.set_result({"label": "error", "confidence": 0.0, "scores": {}, "frames_analyzed": 0, "media_type": mt, "error": f"file_path not found: {fp!r}"})
                futures.append(fut)
                filenames.append(fname)
                continue

            with open(fp, "rb") as f:
                contents = f.read()

            frames = decode_by_media_type(contents, mt)
            if not frames:
                fut = asyncio.get_event_loop().create_future()
                fut.set_result({"label": "error", "confidence": 0.0, "scores": {}, "frames_analyzed": 0, "media_type": mt})
                futures.append(fut)
            else:
                req = InferenceRequest(frames)
                await request_queue.put(req)
                futures.append(req.future)
            filenames.append(fname)
    else:
        for file, mt in zip(files, media_types):
            contents = await file.read()
            try:
                _check_file_size(contents, file.filename)
            except HTTPException as e:
                fut = asyncio.get_event_loop().create_future()
                fut.set_exception(e)
                futures.append(fut)
                filenames.append(file.filename or "unknown")
                continue

            frames = decode_by_media_type(contents, mt)
            if not frames:
                fut = asyncio.get_event_loop().create_future()
                fut.set_result({"label": "error", "confidence": 0.0, "scores": {}, "frames_analyzed": 0, "media_type": mt})
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
            res = {"label": "error", "confidence": 0.0, "error": str(e), "frames_analyzed": 0, "media_type": media_types[i]}
        res["filename"] = filenames[i]
        res["image_index"] = i
        results.append(res)

    total_time = time.time() - start
    return JSONResponse(content={
        "results": results,
        "total_files": source_count,
        "total_processing_time_ms": round(total_time * 1000, 2),
        "avg_time_per_file_ms": round((total_time / max(source_count, 1)) * 1000, 2),
    })


if sys.platform != "win32":
    if sys.version_info >= (3, 11):
        asyncio.DefaultEventLoopPolicy = uvloop.EventLoopPolicy
    else:
        uvloop.install()

if __name__ == "__main__":
    import multiprocessing
    workers = 1 if torch.cuda.is_available() else min(multiprocessing.cpu_count(), 4)
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", "8000")),
        workers=workers,
        loop="uvloop",
        log_level="info",
        access_log=True,
    )
