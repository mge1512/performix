# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

import time
import logging
from .writer import Operation, Dispatch, Writer
from typing import Any


def time_monotnonic_raw() -> int:
    return time.clock_gettime_ns(time.CLOCK_MONOTONIC_RAW)


class OperationTracker:
    operation: Operation | None = None
    dispatch: Dispatch | None = None

    def __init__(self, writer: Writer):
        self.writer = writer

    def begin_operation(self, name: str, args: list[Any], kwargs: dict[str, Any]):
        """
        Track the beginning of a pytorch operation.
        """
        if self.operation is not None:
            logging.warning("Resetting tracked operation!")

        self.operation = Operation(name)
        self.operation.timestamp_begin = time_monotnonic_raw()
        self.operation.args = args
        self.operation.kwargs = kwargs

    def end_operation(self, output: Any):
        """
        Track the end of a pytorch operation.
        """
        if self.operation is None:
            logging.warning("No tracked operation!")
            return

        self.operation.timestamp_end = time_monotnonic_raw()
        self.operation.output = output
        self.writer.write_operation(self.operation)
        self.operation = None

    def begin_dispatch(self, name: str, args: list[Any], kwargs: dict[str, Any]):
        """
        Track the beginning of a pytorch operation dispatch.
        """
        if self.operation is None:
            logging.warning("No tracked operation!")
            return

        if self.dispatch is not None:
            logging.warning("Resetting tracked dispatch!")

        self.dispatch = Dispatch(name)
        self.dispatch.timestamp_begin = time_monotnonic_raw()
        self.dispatch.args = args
        self.dispatch.kwargs = kwargs

    def end_dispatch(self, output: Any):
        """
        Track the end of a pytorch operation dispatch.
        """
        if self.operation is None:
            logging.warning("No tracked operation!")
            return

        if self.dispatch is None:
            logging.warning("No tracked dispatch!")
            return

        self.dispatch.timestamp_end = time_monotnonic_raw()
        self.dispatch.output = output
        self.operation.dispatch.append(self.dispatch)
        self.dispatch = None
