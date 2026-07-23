from __future__ import annotations

from qst.ir import validate_ir_v04
from qst.tokens import validate_token_maturity_for_profile
from tests.token_acceptance._helpers import spec_by_name, strategy_for


def test_stage3b_maturity_profile_matrix_is_enforced_by_validator() -> None:
    specs = spec_by_name()

    accepted = specs["math.add"]
    frozen = accepted.model_copy(
        update={"surface": accepted.surface.model_copy(update={"maturity": "frozen"})}
    )
    experimental = specs["score.calibrate"]
    reserved = specs["event.filter"]
    deprecated = accepted.model_copy(
        update={"surface": accepted.surface.model_copy(update={"maturity": "deprecated"})}
    )

    for profile in ("research", "paper", "pretrade", "production_guarded"):
        assert validate_token_maturity_for_profile(accepted, profile=profile) == []  # type: ignore[arg-type]
        assert validate_token_maturity_for_profile(frozen, profile=profile) == []  # type: ignore[arg-type]
        reserved_result = validate_ir_v04(strategy_for(reserved.token_ref.name), profile=profile)  # type: ignore[arg-type]
        assert not reserved_result.ok
        assert reserved_result.errors[0].code == "QST_TOKEN_RESERVED_DESIGN_NOT_EXECUTABLE"

    assert validate_token_maturity_for_profile(experimental, profile="research")[0].severity == "warning"
    assert validate_token_maturity_for_profile(experimental, profile="paper")[0].severity == "warning"
    assert validate_token_maturity_for_profile(experimental, profile="pretrade")[0].severity == "error"
    assert (
        validate_token_maturity_for_profile(experimental, profile="production_guarded")[0].severity
        == "error"
    )
    assert validate_token_maturity_for_profile(deprecated, profile="research")[0].severity == "warning"
    assert validate_token_maturity_for_profile(deprecated, profile="paper")[0].severity == "warning"
    assert validate_token_maturity_for_profile(deprecated, profile="pretrade")[0].severity == "warning"
    assert (
        validate_token_maturity_for_profile(deprecated, profile="production_guarded")[0].severity
        == "error"
    )


def test_stage3b_profile_gate_diagnostic_codes_are_stable() -> None:
    specs = spec_by_name()
    experimental = validate_token_maturity_for_profile(specs["score.calibrate"], profile="pretrade")
    deprecated = specs["math.add"].model_copy(
        update={"surface": specs["math.add"].surface.model_copy(update={"maturity": "deprecated"})}
    )
    deprecated_diagnostics = validate_token_maturity_for_profile(
        deprecated,
        profile="production_guarded",
    )

    assert experimental[0].code == "QST_TOKEN_EXPERIMENTAL_PROFILE_GATE"
    assert deprecated_diagnostics[0].code == "QST_TOKEN_DEPRECATED_PROFILE_GATE"
