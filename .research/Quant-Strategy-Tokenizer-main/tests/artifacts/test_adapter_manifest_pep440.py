from __future__ import annotations

import pytest
from pydantic import ValidationError

from qst.artifacts import AdapterManifest
from tests.artifacts.schema_helpers import validate_schema


def test_adapter_manifest_schema_and_pep440_validation() -> None:
    manifest = AdapterManifest(
        adapter_id="mock.backtest",
        adapter_version="0.4.0",
        qst_core_compatible=">=0.4,<0.5",
        implements_ports=["backtest"],
        supported_engines=["mock"],
    )

    validate_schema("adapter_manifest-0.4.schema.json", manifest.model_dump(mode="json"))


def test_adapter_manifest_rejects_non_pep440_values() -> None:
    with pytest.raises(ValidationError, match="PEP 440 version"):
        AdapterManifest(
            adapter_id="mock.backtest",
            adapter_version="v0.4.0-p4",
            qst_core_compatible=">=0.4,<0.5",
            implements_ports=["backtest"],
        )

    with pytest.raises(ValidationError, match="PEP 440 SpecifierSet"):
        AdapterManifest(
            adapter_id="mock.backtest",
            adapter_version="0.4.0",
            qst_core_compatible="not a specifier",
            implements_ports=["backtest"],
        )
