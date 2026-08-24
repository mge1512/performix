# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

import logging
from unittest.mock import patch

from pytorch_collect.tracker import OperationTracker
from pytorch_collect.writer import Writer


class MockWriter(Writer):
    def __init__(self):
        self.operations = []

    def write_operation(self, operation):
        self.operations.append(operation)


def assert_warning_logged(caplog, message):
    assert any(
        record.levelno == logging.WARNING and record.message == message
        for record in caplog.records
    )


def test_records_operation_with_dispatch_and_timestamps():
    writer = MockWriter()
    tracker = OperationTracker(writer)

    with patch("pytorch_collect.tracker.time_monotnonic_raw", side_effect=[100, 110, 150, 200]):
        tracker.begin_operation("add", [{"type": "Tensor"}], {
                                "alpha": {"value": 1}})
        tracker.begin_dispatch("aten.add.Tensor", [{"type": "Tensor"}], {
                               "alpha": {"value": 1}})
        tracker.end_dispatch({"type": "Tensor"})
        tracker.end_operation({"type": "Tensor"})

    assert len(writer.operations) == 1

    operation = writer.operations[0]
    assert operation.name == "add"
    assert operation.timestamp_begin == 100
    assert operation.timestamp_end == 200
    assert operation.args == [{"type": "Tensor"}]
    assert operation.kwargs == {"alpha": {"value": 1}}
    assert operation.output == {"type": "Tensor"}
    assert len(operation.dispatch) == 1

    dispatch = operation.dispatch[0]
    assert dispatch.name == "aten.add.Tensor"
    assert dispatch.timestamp_begin == 110
    assert dispatch.timestamp_end == 150
    assert dispatch.args == [{"type": "Tensor"}]
    assert dispatch.kwargs == {"alpha": {"value": 1}}
    assert dispatch.output == {"type": "Tensor"}


def test_begin_operation_replaces_existing_operation_and_warns(caplog):
    writer = MockWriter()
    tracker = OperationTracker(writer)

    with patch("pytorch_collect.tracker.time_monotnonic_raw", side_effect=[100, 200]):
        tracker.begin_operation("old", [], {})
        tracker.begin_operation("new", [{"type": "Tensor"}], {})

    assert_warning_logged(caplog, "Resetting tracked operation!")
    assert tracker.operation.name == "new"
    assert tracker.operation.timestamp_begin == 200
    assert tracker.operation.args == [{"type": "Tensor"}]
    assert tracker.operation.kwargs == {}
    assert writer.operations == []


def test_end_operation_without_operation_warns_and_does_not_write(caplog):
    writer = MockWriter()
    tracker = OperationTracker(writer)

    tracker.end_operation({"type": "Tensor"})

    assert_warning_logged(caplog, "No tracked operation!")
    assert writer.operations == []


def test_begin_dispatch_without_operation_warns_and_does_not_start_dispatch(caplog):
    tracker = OperationTracker(MockWriter())

    tracker.begin_dispatch("aten.add.Tensor", [], {})

    assert_warning_logged(caplog, "No tracked operation!")
    assert tracker.dispatch is None


def test_end_dispatch_without_operation_warns(caplog):
    writer = MockWriter()
    tracker = OperationTracker(writer)

    tracker.end_dispatch({"type": "Tensor"})

    assert_warning_logged(caplog, "No tracked operation!")
    assert tracker.dispatch is None
    assert writer.operations == []


def test_end_dispatch_without_dispatch_warns_and_keeps_operation(caplog):
    writer = MockWriter()
    tracker = OperationTracker(writer)

    with patch("pytorch_collect.tracker.time_monotnonic_raw", return_value=100):
        tracker.begin_operation("add", [], {})

    tracker.end_dispatch({"type": "Tensor"})

    assert_warning_logged(caplog, "No tracked dispatch!")
    assert tracker.operation.name == "add"
    assert tracker.operation.dispatch == []
    assert writer.operations == []
