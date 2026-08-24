// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package jstest

import (
	"reflect"

	"github.com/Arm-Debug/apap-cli/apap-engine/gojautils"
	tool_goja "github.com/Arm-Debug/apap-cli/apap-engine/tool/goja"
)

// DecodesTo returns a predicate function that in turn returns
// true if its argument can be converted to type T and equals
// `expected`.
//
// This is a helper function to be used alongside mock.MatchedBy,
// which allows matching a value of a param with a broad type
// (like any, []any, map[string]any etc.) against a concrete
// expected value.
func DecodesTo[T any](expected T) func(any) bool {
	return func(actual any) bool {
		var decoded T
		err := gojautils.ParseObjectWithRegex(actual, &decoded, nil, nil)
		return err == nil && reflect.DeepEqual(expected, decoded)
	}
}

// EmptyToolContext returns a ToolContext struct with 0-initialized
// values. Use this instead of manually constructing an empty
// ToolContext if its fields will be accessed in JS, otherwise some
// fields such as `params` will be nil/undefined.
//
// Note that Workload will still be nil because this doesn't have a
// non-nil empty value.
func EmptyToolContext() tool_goja.ToolContext {
	return tool_goja.ToolContext{
		Params:     map[string]any{},
		Workload:   nil,
		WorkingDir: "",
		Env:        map[string]string{},
		Timeout:    0,
		ToolsRoot:  "",
		Metadata:   map[string]any{},
	}
}
