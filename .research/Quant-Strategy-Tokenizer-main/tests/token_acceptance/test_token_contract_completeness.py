from __future__ import annotations

from tests.token_acceptance._helpers import REPORT_ROOT, all_specs


def test_stage3b_accepted_tokens_have_contracts_sufficient_for_validation() -> None:
    for spec in all_specs():
        surface = spec.surface
        contract = surface.contract

        assert contract.temporal
        assert contract.numeric
        assert contract.missing_data
        assert contract.failure_mode
        assert contract.supported_profiles == (
            "research",
            "paper",
            "pretrade",
            "production_guarded",
        )
        if surface.maturity == "accepted":
            assert surface.execution_support in {
                "metadata_only",
                "reference_helper",
                "runtime_executor",
                "external_only",
            }
            assert contract.scope in {"validation_only", "reference_semantics", "execution_semantics"}


def test_stage3b_specialized_contracts_are_not_silent_annotations() -> None:
    for spec in all_specs():
        surface = spec.surface
        contract = surface.contract

        if surface.capabilities.stateful:
            assert contract.state
        if surface.capabilities.panel_aware:
            assert contract.panel
        if surface.capabilities.solver_backed:
            assert contract.solver is not None
            assert contract.solver.solver_required is True
            assert contract.solver.bit_exact_claim is False
            assert surface.execution_support == "metadata_only"
        if surface.family == "window":
            temporal = contract.temporal
            assert any(marker in temporal for marker in ("window", "min_history", "trailing"))
        if surface.maturity == "reserved_design":
            assert surface.execution_support == "metadata_only"
            assert contract.scope == "validation_only"
            assert surface.capabilities.reserved_only is True


def test_stage3b_contract_audit_report_records_no_repair_blocker() -> None:
    report = (REPORT_ROOT / "token_contract_audit.md").read_text(encoding="utf-8")

    assert "The Stage 3A token contracts are accepted." in report
    assert "No accepted token requires repair before Stage 3B acceptance." in report
