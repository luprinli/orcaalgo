from __future__ import annotations

import builtins
import json
import tempfile
from pathlib import Path

import numpy as np
import pytest

from orca.ml.inference import _predict_json_model

_original_import = builtins.__import__


def _block_xgboost_import(name, *args, **kwargs):
    if name == "xgboost":
        raise ImportError("blocked for testing fallback path")
    return _original_import(name, *args, **kwargs)


@pytest.fixture(autouse=True)
def _force_fallback(monkeypatch):
    monkeypatch.setattr(builtins, "__import__", _block_xgboost_import)


class TestPredictJSONModel:
    def test_logistic_with_importances(self):
        model_data = {
            "feature_importances": [0.1] * 21,
            "metadata": {"version": "v1"},
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X = np.array([[0.5] * 21], dtype=np.float64)
            p = _predict_json_model(str(path), X)

            assert 0.0 <= p <= 1.0

    def test_negative_features(self):
        model_data = {"feature_importances": [0.1] * 21}
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X = np.array([[-1.0] * 21], dtype=np.float64)
            p = _predict_json_model(str(path), X)

            assert 0.0 <= p <= 1.0

    def test_output_is_clamped(self):
        model_data = {"feature_importances": [100.0] + [0.0] * 20}
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X = np.array([[10.0] + [0.0] * 20], dtype=np.float64)
            p = _predict_json_model(str(path), X)

            assert 0.0 <= p <= 1.0

    def test_feature_count_mismatch(self):
        model_data = {"feature_importances": [0.1] * 5}
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X = np.array([[0.5] * 21], dtype=np.float64)
            with pytest.raises(ValueError, match="feature count mismatch"):
                _predict_json_model(str(path), X)

    def test_all_zero_features(self):
        model_data = {"feature_importances": [0.1] * 21}
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X = np.array([[0.0] * 21], dtype=np.float64)
            p = _predict_json_model(str(path), X)

            assert p == pytest.approx(0.5, abs=0.01)

    def test_mixed_importances(self):
        model_data = {
            "feature_importances": [0.5, -0.3] + [0.0] * 19,
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            X_pos = np.array([[1.0, 1.0] + [0.0] * 19], dtype=np.float64)
            X_neg = np.array([[-1.0, 1.0] + [0.0] * 19], dtype=np.float64)

            p_pos = _predict_json_model(str(path), X_pos)
            p_neg = _predict_json_model(str(path), X_neg)

            assert 0.0 <= p_pos <= 1.0
            assert 0.0 <= p_neg <= 1.0
            assert p_pos > p_neg

    def test_platt_scaling_applied(self):
        model_data = {
            "feature_importances": [0.5, 0.3] + [0.0] * 19,
            "platt_a": 2.0,
            "platt_b": -1.0,
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            from orca.ml.inference import _predict_json_model

            X = np.array([[1.0, 1.0] + [0.0] * 19], dtype=np.float64)
            p = _predict_json_model(str(path), X)
            assert 0.0 <= p <= 1.0

    def test_platt_identity(self):
        model_data = {
            "feature_importances": [0.5] + [0.0] * 20,
            "platt_a": 1.0,
            "platt_b": 0.0,
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            from orca.ml.inference import _predict_json_model

            X = np.array([[1.0] + [0.0] * 20], dtype=np.float64)
            p = _predict_json_model(str(path), X)
            assert 0.5 < p < 0.7

    def test_platt_without_params(self):
        model_data = {"feature_importances": [0.5] + [0.0] * 20}
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "model.json"
            path.write_text(json.dumps(model_data))

            from orca.ml.inference import _predict_json_model

            X = np.array([[1.0] + [0.0] * 20], dtype=np.float64)
            p = _predict_json_model(str(path), X)
            assert 0.0 <= p <= 1.0
