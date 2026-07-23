from __future__ import annotations

from qst.ir import validate_ir_v04
from tests.token_acceptance._helpers import spec_by_name, strategy_for

RESERVED_TOKENS = {
    "event.join_asof",
    "event.filter",
    "event.window_count",
    "distribution.normal_fit",
    "distribution.quantile",
    "distribution.tail_probability",
    "execution.submit_order",
    "execution.cancel_order",
    "execution.fill_report",
}


def test_stage3b_reserved_tokens_are_visible_but_non_executable() -> None:
    specs = spec_by_name()

    assert RESERVED_TOKENS <= set(specs)
    for name in sorted(RESERVED_TOKENS):
        surface = specs[name].surface
        assert surface.maturity == "reserved_design"
        assert surface.execution_support == "metadata_only"
        assert surface.contract.scope == "validation_only"
        assert surface.capabilities.reserved_only is True
        assert surface.capabilities.deterministic_level == "reserved"


def test_stage3b_reserved_tokens_fail_validation_in_every_profile() -> None:
    for name in sorted(RESERVED_TOKENS):
        for profile in ("research", "paper", "pretrade", "production_guarded"):
            result = validate_ir_v04(strategy_for(name), profile=profile)  # type: ignore[arg-type]

            assert not result.ok
            assert result.errors[0].code == "QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE"
            assert result.errors[0].severity == "error"
            assert result.errors[0].profile == profile
