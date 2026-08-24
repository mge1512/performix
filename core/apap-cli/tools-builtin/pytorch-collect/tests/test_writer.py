# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

import os
from pytorch_collect.writer import Dispatch, Operation, ParquetWriter, PrintWriter
import pyarrow.parquet as pq


def test_print_writer_closes_dispatch_timestamp_parenthesis(capsys):
    operation = Operation(
        name="torch.add",
        dispatch=[Dispatch(name="aten.add.Tensor", timestamp_begin=123, timestamp_end=456)],
    )

    PrintWriter().write_operation(operation)

    assert "[dispatch] aten.add.Tensor @(123, 456)" in capsys.readouterr().out


def test_parquet_schema_version(tmp_path):
    """
    Assert on the current version of the parquet schema (ensure it matches the one in the README!)
    """
    with ParquetWriter(tmp_path, 1024):
        pass

    operations = pq.read_metadata(tmp_path / "operations.parquet").metadata
    dispatches = pq.read_metadata(tmp_path / "dispatches.parquet").metadata

    assert operations[b'schema_version'] == b'1'
    assert dispatches[b'schema_version'] == b'1'


def test_parquet_files_are_always_created(tmp_path):
    with ParquetWriter(tmp_path, 1024):
        pass

    assert os.path.exists(tmp_path / "operations.parquet")
    assert os.path.exists(tmp_path / "dispatches.parquet")


def test_parquet_rows_are_writen(tmp_path):
    with ParquetWriter(tmp_path, 1024) as writer:
        op = Operation(
            name="torch.add",
            timestamp_begin=1000,
            timestamp_end=1500,
            dispatch=[
                Dispatch(
                    name="aten.add.Tensor",
                    timestamp_begin=1050,
                    timestamp_end=1450,
                    args=[
                        {"type": "Tensor", "dtype": "float64",
                            "shape": [4, 4]},
                        {"type": "Tensor", "dtype": "float64",
                            "shape": [4, 4]},
                    ],
                    kwargs={"beta": 2},
                    output={"type": "Tensor",
                            "dtype": "float64", "shape": [4, 4]},
                ),
            ],
            args=[
                {"type": "Tensor", "dtype": "float32", "shape": [2, 2]},
                {"type": "Tensor", "dtype": "float32", "shape": [2, 2]},
            ],
            kwargs={"alpha": 1},
            output={"type": "Tensor", "dtype": "float32", "shape": [2, 2]},
        )

        writer.write_operation(op)

    operations = pq.read_table(tmp_path / "operations.parquet").to_pylist()
    dispatches = pq.read_table(tmp_path / "dispatches.parquet").to_pylist()

    assert operations == [
        {
            "op_id": 0,
            "op_name": "torch.add",
            "ts_begin_ns": 1000,
            "ts_end_ns": 1500,
            "dispatch_count": 1,
            "args_json": '[{"dtype": "float32", "shape": [2, 2], "type": "Tensor"}, {"dtype": "float32", "shape": [2, 2], "type": "Tensor"}]',
            "kwargs_json": '{"alpha": 1}',
            "output_json": '{"dtype": "float32", "shape": [2, 2], "type": "Tensor"}',
        }
    ]

    assert dispatches == [
        {
            "op_id": 0,
            "dispatch_id": 0,
            "dispatch_name": "aten.add.Tensor",
            "ts_begin_ns": 1050,
            "ts_end_ns": 1450,
            "args_json": '[{"dtype": "float64", "shape": [4, 4], "type": "Tensor"}, {"dtype": "float64", "shape": [4, 4], "type": "Tensor"}]',
            "kwargs_json": '{"beta": 2}',
            "output_json": '{"dtype": "float64", "shape": [4, 4], "type": "Tensor"}',
        }
    ]
