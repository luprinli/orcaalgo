from __future__ import annotations

from pathlib import Path

from qst.hash import compute_hashes_v2
from qst.ir import load_ir_v04_file

ROOT = Path(__file__).resolve().parents[2]


def test_current_reference_strategy_hashes_are_stable() -> None:
    cases = {
        "examples/strategies/kdj_cross_basic.gkr.yaml": {
            "graph_hash": "sha256:34822b0de8b9b517c3b5cdb04f79adb2f9cd30ee63833891e62c204f6034411e",
            "param_hash": "sha256:18f04bf380d53ad8020680f74c5686ba6dcf569543b4551b3b597f13aca6bb5c",
            "instance_hash": "sha256:56fd90013048a81f9be6e2bc13adbf732c23f01c275a05ff598f6f9b9df67f25",
        },
        "examples/strategies/kdj_with_ema_filter.gkr.yaml": {
            "graph_hash": "sha256:c4ac597df9553a45363832e1f9f919fb4948856dd5964191d5cdab7f2b058f8f",
            "param_hash": "sha256:078d1dcccd7322b55efb91adf6485f0b868e07e4588ecc0564bea31200b487b0",
            "instance_hash": "sha256:8fecad9fed31671da2ca8406a3ecebf0da7da2067865dbb9004807138f9d647b",
        },
    }

    for relative, expected in cases.items():
        hashes = compute_hashes_v2(load_ir_v04_file(ROOT / relative))
        assert hashes.__dict__ == expected
