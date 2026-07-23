"""ONNX model export for Go inference runtime.

Converts trained XGBoost models to ONNX format via sklearn-onnx for
cross-platform inference. The ONNX format enables inference in Go via
onnxruntime-go or as a fallback subprocess call to Python.
"""

from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from pathlib import Path

logger = logging.getLogger("orca.ml.train.export_onnx")


@dataclass(frozen=True)
class ONNXExportResult:
    onnx_path: str
    input_name: str
    output_name: str
    input_shape: tuple[int, ...]
    output_shape: tuple[int, ...]
    metadata_path: str


def export_to_onnx(
    model,
    output_dir: str | Path,
    model_name: str = "meta_labeling",
    n_features: int = 21,
    initial_types: list | None = None,
) -> ONNXExportResult:
    """Export a trained XGBoost model to ONNX format.

    Args:
        model: Trained XGBoost model (XGBClassifier or XGBRegressor).
        output_dir: Output directory for ONNX and metadata files.
        model_name: Base name for the model files.
        n_features: Number of input features.
        initial_types: Optional initial type specification.

    Returns:
        ONNXExportResult with paths and metadata.
    """
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    onnx_path = output_dir / f"{model_name}.onnx"
    metadata_path = output_dir / f"{model_name}_metadata.json"

    try:
        from skl2onnx import convert_sklearn
        from skl2onnx.common.data_types import FloatTensorType

        if initial_types is None:
            initial_types = [("float_input", FloatTensorType([None, n_features]))]

        onnx_model = convert_sklearn(
            model,
            initial_types=initial_types,
            target_opset=12,
        )

        with open(onnx_path, "wb") as f:
            f.write(onnx_model.SerializeToString())

        input_shape = (1, n_features)
        output_shape = (1, 2)  # binary: [p_0, p_1]

        metadata = {
            "model_name": model_name,
            "input_name": initial_types[0][0],
            "input_shape": list(input_shape),
            "output_shape": list(output_shape),
            "n_features": n_features,
            "opset": 12,
            "created_at": str(Path(onnx_path).stat().st_mtime),
        }
        with open(metadata_path, "w") as f:
            json.dump(metadata, f, indent=2)

        logger.info(
            "exported ONNX: %s (%d bytes, %d features)",
            onnx_path, onnx_path.stat().st_size, n_features,
        )

        return ONNXExportResult(
            onnx_path=str(onnx_path),
            input_name=initial_types[0][0],
            output_name="probabilities",
            input_shape=input_shape,
            output_shape=output_shape,
            metadata_path=str(metadata_path),
        )

    except ImportError:
        logger.warning("skl2onnx not installed, exporting as JSON predictor instead")
        return _export_as_json_predictor(model, output_dir, model_name, n_features)


def _export_as_json_predictor(
    model,
    output_dir: Path,
    model_name: str,
    n_features: int,
) -> ONNXExportResult:
    """Fallback: export model parameters as JSON for simple Go inference.

    Extracts tree structure and thresholds for a basic Go-native predictor.
    """
    json_path = output_dir / f"{model_name}.json"
    metadata_path = output_dir / f"{model_name}_metadata.json"

    model_data = {
        "model_type": "xgboost",
        "n_features": n_features,
        "n_estimators": model.n_estimators,
        "feature_importances": model.feature_importances_.tolist(),
        "classes": model.classes_.tolist(),
    }

    try:
        booster = model.get_booster()
        model_data["model_dump"] = booster.get_dump(dump_format="json")
        with open(json_path, "w") as f:
            json.dump(model_data, f)

        logger.info("exported JSON predictor: %s", json_path)
    except Exception as e:
        logger.error("JSON export failed: %s", e)
        raise

    metadata = {
        "model_name": model_name,
        "input_shape": [1, n_features],
        "output_shape": [1, 2],
        "n_features": n_features,
        "format": "json",
    }
    with open(metadata_path, "w") as f:
        json.dump(metadata, f, indent=2)

    return ONNXExportResult(
        onnx_path=str(json_path),
        input_name="float_input",
        output_name="probabilities",
        input_shape=(1, n_features),
        output_shape=(1, 2),
        metadata_path=str(metadata_path),
    )
