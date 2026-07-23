from __future__ import annotations

from collections import Counter

from qst.tokens import builtin_token_packs
from tests.token_acceptance._helpers import (
    EXPECTED_FAMILIES,
    EXPECTED_PACK_IDS,
    EXPECTED_TOKEN_COUNT,
    REPORT_ROOT,
    all_specs,
    registry,
)


def test_stage3b_token_inventory_is_complete_and_deterministic() -> None:
    packs = builtin_token_packs()
    specs = all_specs()
    refs = [spec.ref_key for spec in specs]

    assert [pack.pack_id for pack in packs] == EXPECTED_PACK_IDS
    assert len(specs) == EXPECTED_TOKEN_COUNT
    assert Counter(refs).most_common(1)[0][1] == 1
    assert registry().result.ok
    for pack in packs:
        assert [spec.ref_key for spec in pack.tokens] == sorted(spec.ref_key for spec in pack.tokens)


def test_stage3b_all_builtin_tokens_have_public_surface_metadata() -> None:
    specs = all_specs()

    assert {spec.surface.family for spec in specs} == EXPECTED_FAMILIES
    for spec in specs:
        surface = spec.surface
        assert spec.token_ref.namespace == "core"
        assert surface.schema_version == "qst-token-surface/0.4"
        assert surface.family
        assert surface.category
        assert surface.layer
        assert surface.maturity
        assert surface.execution_support
        assert surface.contract
        assert surface.capabilities.deterministic_level


def test_stage3b_surface_acceptance_report_covers_every_builtin_token() -> None:
    report = (REPORT_ROOT / "token_surface_acceptance.md").read_text(encoding="utf-8")

    assert f"Built-in tokens: `{EXPECTED_TOKEN_COUNT}`" in report
    assert "Registry result: `ok`" in report
    for spec in all_specs():
        assert f"| {spec.token_id} |" in report
