"""ML model inference script — called by Go via subprocess.

Reads JSON from stdin: {"model_path": "...", "features": [...]}
Writes JSON to stdout: {"p_win": 0.72, "version": "v1"}
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import numpy as np


def main() -> None:
    try:
        raw = sys.stdin.read()
        if not raw.strip():
            print(json.dumps({"error": "empty input"}))
            sys.exit(1)

        data = json.loads(raw)
        model_path = data.get("model_path", "models/meta_labeling.json")
        features = data.get("features", [])

        if len(features) != 21:
            print(json.dumps({"error": f"expected 21 features, got {len(features)}"}))
            sys.exit(1)

        X = np.array(features, dtype=np.float64).reshape(1, -1)

        if not Path(model_path).exists():
            print(json.dumps({"error": f"model not found: {model_path}"}))
            sys.exit(1)

        p_win = _predict_json_model(model_path, X)

        result = {"p_win": float(p_win), "version": "v1"}
        print(json.dumps(result))

    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


def _predict_json_model(model_path: str, X: np.ndarray) -> float:
    with open(model_path) as f:
        model_data = json.load(f)

    platt_a = model_data.get("platt_a", 1.0)
    platt_b = model_data.get("platt_b", 0.0)

    try:
        import xgboost as xgb
        model = xgb.XGBClassifier()
        model.load_model(model_path)
        proba = model.predict_proba(X)
        p_raw = float(proba[0, 1])
        if platt_a != 1.0 or platt_b != 0.0:
            p_raw = max(min(p_raw, 0.9999), 0.0001)
            logit = float(np.log(p_raw / (1.0 - p_raw)))
            p_cal = 1.0 / (1.0 + np.exp(-(platt_a * logit + platt_b)))
            return max(0.0, min(1.0, p_cal))
        return max(0.0, min(1.0, p_raw))
    except ImportError:
        pass

    importances = model_data.get("feature_importances", [])
    if len(importances) != X.shape[1]:
        raise ValueError(
            f"feature count mismatch: model={len(importances)} input={X.shape[1]}"
        )

    score = float(np.dot(X[0], importances))
    p_win = 1.0 / (1.0 + np.exp(-(platt_a * score + platt_b)))
    return max(0.0, min(1.0, p_win))


if __name__ == "__main__":
    main()
