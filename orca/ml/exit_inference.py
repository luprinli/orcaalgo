"""Exit optimization inference script — called by Go via subprocess.

Reads JSON from stdin: {"model_path": "...", "features": [...]}
Writes JSON to stdout: {"urgency": 0.35, "version": "v1"}
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
        model_path = data.get("model_path", "models/exit_model.json")
        features = data.get("features", [])

        if len(features) != 12:
            print(json.dumps({"error": f"expected 12 features, got {len(features)}"}))
            sys.exit(1)

        X = np.array(features, dtype=np.float64).reshape(1, -1)
        urgency = _predict(model_path, X)

        result = {"urgency": float(np.clip(urgency, 0.0, 1.0)), "version": "v1"}
        print(json.dumps(result))

    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


def _predict(model_path: str, X: np.ndarray) -> float:
    if not Path(model_path).exists():
        return 0.5

    try:
        import lightgbm as lgb

        model = lgb.Booster(model_file=model_path)
        return float(model.predict(X)[0])
    except (ImportError, Exception):
        pass

    try:
        import xgboost as xgb

        model = xgb.XGBRegressor()
        model.load_model(model_path)
        return float(model.predict(X)[0])
    except (ImportError, Exception):
        pass

    return 0.5


if __name__ == "__main__":
    main()
