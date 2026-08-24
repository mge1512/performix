<!--
# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0
-->

# PyTorch Collect

PyTorch Collect traces a Python workload that uses PyTorch and records:

- high-level PyTorch operations.
- dispatcher calls to low-level tensor operations.
- start and end timestamps for each operation and dispatch (using CLOCK_MONOTONIC_RAW).
- compact JSON summaries of dispatch arguments, keyword arguments, and outputs.

## Install

Install the package locally from this directory:

```bash
pip install "."
pip install ".[samples]" # optional to run local sample workloads
```

## Usage

Run a Python module file under the collector, including any arguments used by the module:

```bash
pytorch-collect --output ./trace-output --writer parquet -- samples/gpt2.py "What is Arm Performix?" 100
```

This generates two parquet files under the trace-output directory:

- `operations.parquet`
- `dispatches.parquet`

## Parquet Schema

Current schema version: `1`

### `operations.parquet`

| Column | Type | Description |
| --- | --- | --- |
| `op_id` | `int64` | Operation ID. |
| `op_name` | `string` | PyTorch operation name. |
| `ts_begin_ns` | `int64` | Operation start time (CLOCK_MONOTONIC_RAW). |
| `ts_end_ns` | `int64` | Operation end time (CLOCK_MONOTONIC_RAW). |
| `dispatch_count` | `int32` | Number of dispatcher calls recorded for the operation. |
| `args_json` | `string` | JSON summary of positional arguments. |
| `kwargs_json` | `string` | JSON summary of keyword arguments. |
| `output_json` | `string` | JSON summary of the dispatcher output. |

### `dispatches.parquet`

| Column | Type | Description |
| --- | --- | --- |
| `op_id` | `int64` | Operation ID (can be joined with operations.parquet column of the same name). |
| `dispatch_id` | `int64` | Dispatcher call ID. |
| `dispatch_name` | `string` | Dispatcher function name. |
| `ts_begin_ns` | `int64` | Dispatcher start time (CLOCK_MONOTONIC_RAW). |
| `ts_end_ns` | `int64` | Dispatcher end time (CLOCK_MONOTONIC_RAW). |
| `args_json` | `string` | JSON summary of positional arguments. |
| `kwargs_json` | `string` | JSON summary of keyword arguments. |
| `output_json` | `string` | JSON summary of the dispatcher output. |

## Testing

In order to run the unit tests, run the following:

```bash
# install dependencies
pip install "."
pip install pytest

# run tests
pytest tests/*
```