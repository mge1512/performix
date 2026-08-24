# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

from torch.overrides import TorchFunctionMode
from torch.utils._python_dispatch import TorchDispatchMode
from torch import Tensor
from .tracker import OperationTracker


class OperatorTrace(TorchFunctionMode):
    def __init__(self, tracker: OperationTracker, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.tracker = tracker

    def __torch_function__(self, func, types, args=(), kwargs=None):
        kwargs = kwargs or {}
        self.tracker.begin_operation(
            func.__name__,
            _extract_args_specs(args),
            _extract_kwargs_specs(kwargs)
        )

        try:
            output = func(*args, **kwargs)

            self.tracker.end_operation(_extract_output_specs(output))
        except Exception as ex:
            self.tracker.end_operation(None)
            raise ex

        return output


class DispatchTrace(TorchDispatchMode):
    def __init__(self, tracker: OperationTracker, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.tracker = tracker

    def __torch_dispatch__(self, func, types, args=(), kwargs=None):
        kwargs = kwargs or {}
        self.tracker.begin_dispatch(
            str(func),
            _extract_args_specs(args),
            _extract_kwargs_specs(kwargs)
        )

        try:
            output = func(*args, **kwargs)

            self.tracker.end_dispatch(_extract_output_specs(output))
        except Exception as ex:
            self.tracker.end_dispatch(None)
            raise ex

        return output


def _extract_arg_spec(arg):
    spec = {
        'type': type(arg).__name__,
    }

    if isinstance(arg, Tensor):
        spec['shape'] = list(arg.shape)
        spec['dtype'] = str(arg.dtype)
    elif isinstance(arg, bool | int | float | str):
        spec['value'] = arg
    elif isinstance(arg, tuple | list):
        spec['values'] = _extract_args_specs(arg)

    return spec


def _extract_args_specs(args):
    return [_extract_arg_spec(arg) for arg in args]


def _extract_kwargs_specs(kwargs):
    return {key: _extract_arg_spec(arg) for key, arg in kwargs.items()}


def _extract_output_specs(output):
    return _extract_arg_spec(output)
