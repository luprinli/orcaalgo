from __future__ import annotations

from qst.ir import is_gkr_package, is_gkr_source


def test_gkr_source_suffix_detection() -> None:
    assert is_gkr_source("a.gkr.yaml")
    assert not is_gkr_source("a.yaml")
    assert not is_gkr_source("a.gkr")


def test_gkr_package_suffix_detection() -> None:
    assert is_gkr_package("a.gkr")
    assert not is_gkr_package("a.gkr.yaml")
