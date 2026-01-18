from optimum.onnxruntime import ORTModelForImageClassification
from transformers import ViTImageProcessor

MODEL_ID = "Falconsai/nsfw_image_detection"
OUT_DIR = "onnx_model"

# Export PyTorch → ONNX
model = ORTModelForImageClassification.from_pretrained(
    MODEL_ID,
    export=True
)

model.save_pretrained(OUT_DIR)

# Save image processor (MANDATORY)
processor = ViTImageProcessor.from_pretrained(MODEL_ID)
processor.save_pretrained(OUT_DIR)

print("ONNX export DONE")