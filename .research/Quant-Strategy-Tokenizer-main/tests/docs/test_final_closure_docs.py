from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]

FINAL_DOCS = [
    "docs/FINAL_SCOPE.md",
    "docs/FINAL_ACCEPTANCE.md",
    "docs/FINAL_REPORT.md",
    "docs/ROADMAP_CLOSED.md",
    "docs/NON_GOALS.md",
    "docs/FINAL_HANDOFF.md",
]

AGENT_DOCS = [
    "docs/agent/AGENT_TAKEOVER_PROMPT.md",
    "docs/agent/AGENT_PLAYBOOK.md",
    "docs/agent/USAGE_GUIDE.md",
    "docs/agent/TOKEN_REGISTRATION_GUIDE.md",
    "docs/agent/RECIPE_AUTHORING_GUIDE.md",
    "docs/agent/CUSTOM_TOKEN_GUIDE.md",
    "docs/agent/SECONDARY_DEVELOPMENT_GUIDE.md",
]

ADAPTER_DOCS = [
    "docs/adapters/README.md",
    "docs/adapters/QLIB_ADAPTER_BOUNDARY.md",
    "docs/adapters/QLIB_ADAPTER_GUIDE.md",
]

CLOSURE_DOCS = FINAL_DOCS + AGENT_DOCS + ADAPTER_DOCS

FORBIDDEN_STALE_CLAIMS = [
    "quant_strategy_tokenizer",
    ".qst.yaml",
    "qst execute",
    "qst promote",
    "qst compare",
    "qst adapter freqtrade import",
    "freqtrade adapter proof",
]


def _read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def test_final_closure_docs_exist_and_are_non_empty() -> None:
    for relative in CLOSURE_DOCS:
        path = ROOT / relative
        assert path.exists(), relative
        text = path.read_text(encoding="utf-8")
        assert text.startswith("# "), relative
        assert len(text.split()) >= 20, relative


def test_readme_declares_archive_status_and_handoff_links() -> None:
    readme = _read("README.md")
    assert "archived agent-ready research prototype" in readme
    assert "Final Handoff" in readme
    assert "docs/FINAL_HANDOFF.md" in readme
    assert "docs/agent/AGENT_TAKEOVER_PROMPT.md" in readme
    assert "Qlib partial workflow adapter proof" in readme
    assert "qst.cli adapter qlib import" in readme


def test_closure_docs_do_not_claim_stale_package_or_missing_broad_cli() -> None:
    for relative in CLOSURE_DOCS:
        text = _read(relative)
        lowered = text.lower()
        for forbidden in FORBIDDEN_STALE_CLAIMS:
            assert forbidden not in lowered, f"{relative} contains stale claim {forbidden!r}"


def test_adapter_docs_mark_qlib_partial_non_lossless_boundary() -> None:
    joined = "\n".join(_read(relative).lower() for relative in ADAPTER_DOCS)
    for required in [
        "qlib",
        "partial",
        "lossless",
        "broker",
        "exchange",
        "live execution",
        "qrun",
        "model training",
        "backtest",
    ]:
        assert required in joined
    assert "not lossless" in joined or "must not claim lossless" in joined
    assert "qst adapter qlib import" in joined


def test_agent_readme_links_final_handoff_guides_without_replacing_prompt_pack() -> None:
    text = _read("docs/agent/README.md")
    assert "prompts/qst_stage_3c_v0_3_2/README.md" in text
    for relative in [
        "AGENT_TAKEOVER_PROMPT.md",
        "AGENT_PLAYBOOK.md",
        "USAGE_GUIDE.md",
        "TOKEN_REGISTRATION_GUIDE.md",
    ]:
        assert relative in text


def test_qlib_command_is_documented_in_final_acceptance_and_usage() -> None:
    for relative in [
        "docs/FINAL_ACCEPTANCE.md",
        "docs/FINAL_HANDOFF.md",
        "docs/agent/USAGE_GUIDE.md",
    ]:
        text = _read(relative)
        assert "qst.cli adapter qlib import" in text


def test_no_freqtrade_adapter_docs_remain() -> None:
    assert not (ROOT / "docs" / "adapters" / "FREQTRADE_ADAPTER_BOUNDARY.md").exists()
    assert not (ROOT / "docs" / "adapters" / "FREQTRADE_ADAPTER_GUIDE.md").exists()
