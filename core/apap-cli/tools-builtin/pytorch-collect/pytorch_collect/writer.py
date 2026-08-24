# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

from __future__ import annotations

import pyarrow as pa
import pyarrow.parquet as pq
import json
import logging
from dataclasses import dataclass, field
from pathlib import Path
from uuid import uuid4
from typing import Any, override


@dataclass
class Operation:
    name: str
    timestamp_begin: int = 0
    timestamp_end: int = 0
    dispatch: list[Dispatch] = field(default_factory=list)
    args: list[Any] = field(default_factory=list)
    kwargs: dict[str, Any] = field(default_factory=dict)
    output: dict[str, Any] = field(default_factory=dict)


@dataclass
class Dispatch:
    name: str
    timestamp_begin: int = 0
    timestamp_end: int = 0
    args: list[Any] = field(default_factory=list)
    kwargs: dict[str, Any] = field(default_factory=dict)
    output: dict[str, Any] = field(default_factory=dict)


class Writer:
    def __init__(self):
        pass

    def __enter__(self):
        return self

    def __exit__(self, *exception):
        pass

    def write_operation(self, operation: Operation):
        pass


class ParquetWriter(Writer):
    """
    Output the traced pytorch operations to a parquet file.
    """

    schema_version = "1"  # change this value whenever the schema version changes

    @override
    def __init__(self, output: str, flush_limit: int):
        self.output = Path(output)
        self.next_op_id = 0
        self.next_dispatch_id = 0
        self.operation_rows = []
        self.dispatch_rows = []
        self.flush_limit = flush_limit

    @override
    def __enter__(self):
        operations_path, dispatches_path = self._output_paths()
        operations_path.parent.mkdir(parents=True, exist_ok=True)
        dispatches_path.parent.mkdir(parents=True, exist_ok=True)

        self.operations_writer = pq.ParquetWriter(
            operations_path, schema=self._operations_schema(pa).with_metadata({"schema_version": self.schema_version}))
        self.dispatches_writer = pq.ParquetWriter(
            dispatches_path, schema=self._dispatches_schema(pa).with_metadata({"schema_version": self.schema_version}))

        return self

    @override
    def __exit__(self, *exception):
        try:
            self._flush_operations()
            self._flush_dispatches()
        finally:
            self.operations_writer.close()
            self.dispatches_writer.close()

        return False

    @override
    def write_operation(self, op: Operation):
        op_id = self.next_op_id
        self.next_op_id += 1

        self.operation_rows.append({
            "op_id": op_id,
            "op_name": op.name,
            "ts_begin_ns": op.timestamp_begin,
            "ts_end_ns": op.timestamp_end,
            "dispatch_count": len(op.dispatch),
            "args_json": self._to_json(op.args),
            "kwargs_json": self._to_json(op.kwargs),
            "output_json": self._to_json(op.output),
        })

        for dispatch in op.dispatch:
            dispatch_id = self.next_dispatch_id
            self.next_dispatch_id += 1

            self.dispatch_rows.append({
                "op_id": op_id,
                "dispatch_id": dispatch_id,
                "dispatch_name": dispatch.name,
                "ts_begin_ns": dispatch.timestamp_begin,
                "ts_end_ns": dispatch.timestamp_end,
                "args_json": self._to_json(dispatch.args),
                "kwargs_json": self._to_json(dispatch.kwargs),
                "output_json": self._to_json(dispatch.output),
            })

        if len(self.operation_rows) > self.flush_limit:
            self._flush_operations()

        if len(self.dispatch_rows) > self.flush_limit:
            self._flush_dispatches()

    def _flush_operations(self):
        if not self.operation_rows:
            return

        batch = pa.RecordBatch.from_pylist(
            self.operation_rows, schema=self._operations_schema(pa))
        self.operations_writer.write_batch(batch)

        logging.debug(f"Flushed {len(self.operation_rows)} operation rows")
        self.operation_rows.clear()

    def _flush_dispatches(self):
        if not self.dispatch_rows:
            return

        batch = pa.RecordBatch.from_pylist(
            self.dispatch_rows, schema=self._dispatches_schema(pa))
        self.dispatches_writer.write_batch(batch)

        logging.debug(f"Flushed {len(self.dispatch_rows)} dispatch rows")
        self.dispatch_rows.clear()

    def _output_paths(self):
        if self.output.suffix == ".parquet":
            return (
                self.output.with_name(
                    f"{self.output.stem}.operations.parquet"),
                self.output,
            )

        return (
            self.output / "operations.parquet",
            self.output / "dispatches.parquet",
        )

    @staticmethod
    def _operations_schema(pa):
        return pa.schema([
            ("op_id", pa.int64()),
            ("op_name", pa.string()),
            ("ts_begin_ns", pa.int64()),
            ("ts_end_ns", pa.int64()),
            ("dispatch_count", pa.int32()),
            ("args_json", pa.string()),
            ("kwargs_json", pa.string()),
            ("output_json", pa.string()),
        ])

    @staticmethod
    def _dispatches_schema(pa):
        return pa.schema([
            ("op_id", pa.int64()),
            ("dispatch_id", pa.int64()),
            ("dispatch_name", pa.string()),
            ("ts_begin_ns", pa.int64()),
            ("ts_end_ns", pa.int64()),
            ("args_json", pa.string()),
            ("kwargs_json", pa.string()),
            ("output_json", pa.string()),
        ])

    @staticmethod
    def _to_json(value: Any) -> str:
        return json.dumps(value, default=str, sort_keys=True)


class PrintWriter(Writer):
    """
    Output the traced pytorch operations to stdout using print.
    """

    @override
    def write_operation(self, op: Operation):
        build = []
        build.append(
            f"[op] {op.name} @({op.timestamp_begin}, {op.timestamp_end})")
        build.append(f"     args:")

        for arg in op.args:
            build.append(f"         - {arg}")

        build.append(f"     kwargs: {op.kwargs}")
        build.append(f"     output: {op.output}")

        for disp in op.dispatch:
            build.append(
                f"     [dispatch] {disp.name} @({disp.timestamp_begin}, {disp.timestamp_end}")
            build.append(f"                args:")

            for arg in disp.args:
                build.append(f"                    - {arg}")

            build.append(f"                kwargs: {disp.kwargs}")
            build.append(f"                output: {disp.output}")

        print("\n".join(build))
