# Agent Guidance

Agents working in this repository should treat QST as a deterministic record and validation system. The agent should prefer small scoped edits, explicit command evidence, and no hidden trust transitions.

## Active Prompt System

The active prompt system is:

- [qst_stage_3c_v0_3_2](prompts/qst_stage_3c_v0_3_2/README.md)

Validate it with:

```bash
python tools/validate_prompt_set.py docs/agent/prompts/qst_stage_3c_v0_3_2
```

There is no other active prompt system in this tree.

Read these documents first:

- [Methodology](methodology.md)
- [Workflow](workflow.md)
- [Conformance](conformance.md)
- [Task Contract](task_contract.md)
- [Agent Takeover Prompt](AGENT_TAKEOVER_PROMPT.md)
- [Agent Playbook](AGENT_PLAYBOOK.md)
- [Usage Guide](USAGE_GUIDE.md)

Secondary development references:

- [Token Registration Guide](TOKEN_REGISTRATION_GUIDE.md)
- [Recipe Authoring Guide](RECIPE_AUTHORING_GUIDE.md)
- [Custom Token Guide](CUSTOM_TOKEN_GUIDE.md)
- [Secondary Development Guide](SECONDARY_DEVELOPMENT_GUIDE.md)
