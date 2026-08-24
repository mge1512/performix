# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

import argparse
import runpy
import sys
from pathlib import Path
from .tracing import DispatchTrace, OperatorTrace
from .tracker import OperationTracker
from .writer import ParquetWriter, PrintWriter, Writer


def parse_args():
    parser = argparse.ArgumentParser(
        prog="pytorch-collect",
        description="Collect PyTorch operation and dispatch trace data.",
    )
    parser.add_argument(
        "-o",
        "--output",
        default=".",
        help="Directory for collected trace output (for parquet trace writer only).",
    )
    parser.add_argument(
        "-b",
        "--batch",
        default="1024",
        help="Batch size of parquet file writer (for parquet trace writer only).",
    )
    parser.add_argument(
        "--writer",
        choices=("parquet", "print"),
        default="parquet",
        help="Trace writer backend to use.",
    )
    parser.add_argument(
        "module",
        nargs=argparse.REMAINDER,
        help="Python module and optional arguments to run under PyTorch tracing",
    )

    return parser.parse_args()


def trace_module(writer: Writer, module: Path, args: list[str]):
    tracker = OperationTracker(writer)

    with OperatorTrace(tracker), DispatchTrace(tracker):
        module_dir = str(module.parent)

        sys.path = [module_dir, *sys.path]
        sys.argv = [str(module), *args]
        runpy.run_path(str(module), run_name='__main__')


def main():
    args = parse_args()

    if not args.module:
        raise ValueError('module must be specified')

    if args.module[0] == '--':
        args.module.pop(0)

    module = Path(args.module[0])

    if not module.is_file():
        raise ValueError('module must be a file')

    if args.writer == 'parquet':
        with ParquetWriter(args.output, int(args.batch)) as writer:
            trace_module(writer, module, args.module[1:])
    else:
        with PrintWriter() as writer:
            trace_module(writer, module, args.module[1:])
