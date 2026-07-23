from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator
from referencing import Registry, Resource

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_DIR = ROOT / "docs" / "schemas"


def validator_for(schema_name: str) -> Draft202012Validator:
    base_schema = json.loads(
        (SCHEMA_DIR / "artifact_base-0.4.schema.json").read_text(encoding="utf-8")
    )
    schema = json.loads((SCHEMA_DIR / schema_name).read_text(encoding="utf-8"))
    base_id = base_schema["$id"]
    schema_id = schema["$id"]
    registry = Registry().with_resources(
        [
            (base_id, Resource.from_contents(base_schema)),
            (schema_id, Resource.from_contents(schema)),
        ]
    )
    Draft202012Validator.check_schema(base_schema)
    Draft202012Validator.check_schema(schema)
    return Draft202012Validator(schema, registry=registry)


def validate_schema(schema_name: str, payload: dict[str, Any]) -> None:
    validator_for(schema_name).validate(payload)
