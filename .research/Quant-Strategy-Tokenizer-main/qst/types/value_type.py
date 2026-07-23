"""Structured primitive value type names for Token System v2."""

from __future__ import annotations

from typing import Literal

from pydantic import ConfigDict, RootModel

ValueTypeName = Literal[
    "bool",
    "int",
    "float",
    "decimal",
    "string",
    "datetime",
    "json",
    "object",
]


class ValueType(RootModel[ValueTypeName]):
    """A typed primitive value identifier.

    The model validates the value type while preserving the compact canonical
    wire representation used in the manual, for example ``value_type: float``.
    """

    model_config = ConfigDict(frozen=True)

    @property
    def name(self) -> ValueTypeName:
        return self.root
