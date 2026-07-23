"""Regime classifier inference script — called by Go via subprocess.

Reads JSON from stdin: {"model_path": "...", "features": [...]}
Writes JSON to stdout: {"regime_score": 0.82, "regime_state": 2, "probs": [...]}
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import numpy as np

from orca.ml.train.regime_classifier import (
    REGIME_SCORE_WEIGHTS,
    continuous_regime_score,
)


def main() -> None:
    try:
        raw = sys.stdin.read()
        if not raw.strip():
            print(json.dumps({"error": "empty input"}))
            sys.exit(1)

        data = json.loads(raw)
        model_path = data.get("model_path", "models/regime_classifier.json")
        features = data.get("features", [])

        if len(features) != 14:
            print(json.dumps({"error": f"expected 14 features, got {len(features)}"}))
            sys.exit(1)

        X = np.array(features, dtype=np.float64).reshape(1, -1)

        probs, state = _predict(model_path, X)
        score = continuous_regime_score(probs[0], REGIME_SCORE_WEIGHTS)

        result = {
            "regime_score": float(score),
            "regime_state": int(state),
            "probs": [float(p) for p in probs[0]],
            "version": "v1",
        }
        print(json.dumps(result))

    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


def _predict(model_path: str, X: np.ndarray) -> tuple[np.ndarray, int]:
    try:
        import xgboost as xgb
        model = xgb.XGBClassifier()
        model.load_model(model_path)
        probs = model.predict_proba(X)
        state = int(np.argmax(probs[0]))
        return probs, state
    except ImportError:
        pass

    if not Path(model_path).exists():
        raise FileNotFoundError(f"model not found: {model_path}")

    with open(model_path) as f:
        model_data = json.load(f)

    importances = model_data.get("feature_importances", [])
    n_classes = len(model_data.get("classes", [0, 1, 2, 3, 4, 5]))

    if len(importances) > 0:
        logits = np.dot(X, importances[:X.shape[1]])
        state = int(np.argmax(logits))
        probs = np.zeros((1, n_classes))
        probs[0, state] = 1.0
        return probs, state

    state = 1  # default: accumulation
    probs = np.zeros((1, n_classes))
    probs[0, state] = 1.0
    return probs, state


if __name__ == "__main__":
    main()
