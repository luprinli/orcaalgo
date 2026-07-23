"""HMM regime detection training for OrcaAlgo.

Fits a 4-state Gaussian Hidden Markov Model on market return data
and exports calibrated parameters for the Odin runtime engine.
"""

from __future__ import annotations

import json
import math
from dataclasses import asdict, dataclass, field
from pathlib import Path

import numpy as np

__all__ = [
    "HMMParams",
    "export_params_json",
    "export_params_odin",
    "load_params",
    "train_hmm",
]


@dataclass(frozen=True)
class HMMParams:
    n_states: int = 4
    state_labels: list[str] = field(default_factory=lambda: [
        "CALM", "TRENDING", "HIGH_VOL", "CRISIS",
    ])
    transition: list[list[float]] = field(default_factory=list)
    initial_probs: list[float] = field(default_factory=list)
    emission_means: list[float] = field(default_factory=list)
    emission_sds: list[float] = field(default_factory=list)

    def to_json(self, path: str | Path) -> None:
        path = Path(path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(asdict(self), indent=2))

    @classmethod
    def from_dict(cls, data: dict) -> HMMParams:
        return cls(
            n_states=data.get("n_states", 4),
            state_labels=data.get("state_labels", [
                "CALM", "TRENDING", "HIGH_VOL", "CRISIS",
            ]),
            transition=data.get("transition", []),
            initial_probs=data.get("initial_probs", []),
            emission_means=data.get("emission_means", []),
            emission_sds=data.get("emission_sds", []),
        )


def load_params(path: str | Path) -> HMMParams:
    with open(path) as f:
        return HMMParams.from_dict(json.load(f))


def export_params_json(params: HMMParams, path: str | Path = "configs/hmm_params.json") -> HMMParams:
    params.to_json(path)
    return params


def export_params_odin(params: HMMParams, path: str | Path = "odin/risk/hmm_params.odin") -> HMMParams:
    lines = [
        "package risk",
        "",
        "import \"core:math\"",
        "",
        "calibrated_hmm :: proc() -> HMM_Model {",
        "    return HMM_Model{",
        "        transition = [4][4]f64{",
    ]

    for i, row in enumerate(params.transition):
        vals = ", ".join(f"{v:.6f}" for v in row)
        lines.append(f"            {{{vals}}},")

    lines.append("        },")
    lines.append("        initial_probs = [4]f64{")
    prob_str = ", ".join(f"{p:.6f}" for p in params.initial_probs)
    lines.append(f"            {prob_str},")
    lines.append("        },")
    lines.append("        emission_means = [4]f64{")
    means_str = ", ".join(f"{m:.6f}" for m in params.emission_means)
    lines.append(f"            {means_str},")
    lines.append("        },")
    lines.append("        emission_sds = [4]f64{")
    sds_str = ", ".join(f"{s:.6f}" for s in params.emission_sds)
    lines.append(f"            {sds_str},")
    lines.append("        },")
    lines.append("        loaded = true,")
    lines.append("    }")
    lines.append("}")

    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("\n".join(lines) + "\n")
    return params


def _hmm_emission_means() -> np.ndarray:
    return np.array([0.0002, 0.0005, -0.0003, -0.0015])


def _hmm_emission_sds() -> np.ndarray:
    return np.array([0.005, 0.012, 0.025, 0.060])


def _generate_synthetic_returns(
    n_samples: int = 2000,
    seed: int = 42,
) -> np.ndarray:
    rng = np.random.default_rng(seed)
    states = np.zeros(n_samples, dtype=int)
    current = 0
    trans = np.array([
        [0.85, 0.10, 0.04, 0.01],
        [0.08, 0.80, 0.10, 0.02],
        [0.03, 0.10, 0.80, 0.07],
        [0.01, 0.02, 0.10, 0.87],
    ])
    means = _hmm_emission_means()
    sds = _hmm_emission_sds()

    for i in range(1, n_samples):
        current = int(rng.choice(4, p=trans[current]))
        states[i] = current

    returns = np.zeros(n_samples)
    for i in range(n_samples):
        returns[i] = rng.normal(means[states[i]], sds[states[i]])

    return returns


def train_hmm(
    data_path: str,
    n_states: int = 4,
    n_iter: int = 1000,
    seed: int = 42,
) -> HMMParams:
    from hmmlearn.hmm import GaussianHMM

    data_path_obj = Path(data_path)
    if data_path_obj.suffix == ".npy":
        returns = np.load(data_path_obj)
    elif data_path_obj.suffix in (".csv", ".txt"):
        returns = np.loadtxt(data_path_obj)
    else:
        raise ValueError(f"Unsupported data format: {data_path_obj.suffix}")

    returns = np.asarray(returns, dtype=np.float64).reshape(-1, 1)

    model = GaussianHMM(
        n_components=n_states,
        covariance_type="diag",
        n_iter=n_iter,
        random_state=seed,
    )
    model.fit(returns)

    order = np.argsort(model.means_.ravel())
    means_sorted = model.means_[order]
    covars_sorted = model.covars_[order]

    transmat_sorted = model.transmat_[order][:, order]
    startprob_sorted = model.startprob_[order]

    labels = ["CALM", "TRENDING", "HIGH_VOL", "CRISIS"]
    params = HMMParams(
        n_states=n_states,
        state_labels=labels[:n_states],
        transition=transmat_sorted.tolist(),
        initial_probs=startprob_sorted.tolist(),
        emission_means=means_sorted.ravel().tolist(),
        emission_sds=[math.sqrt(float(c)) for c in covars_sorted.ravel().tolist()],
    )

    return params
