// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package jstest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodesTo(t *testing.T) {
	type decodedValue struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	testCases := []struct {
		name     string
		expected decodedValue
		actual   any
		matches  bool
	}{
		{
			name:     "returns true when actual decodes to expected value",
			expected: decodedValue{Name: "a", Count: 1},
			actual:   map[string]any{"name": "a", "count": 1},
			matches:  true,
		},
		{
			name:     "returns false when decoded value differs",
			expected: decodedValue{Name: "a", Count: 1},
			actual:   map[string]any{"name": "a", "count": 2},
			matches:  false,
		},
		{
			name:     "returns false when actual cannot be decoded",
			expected: decodedValue{Name: "a", Count: 1},
			actual:   map[string]any{"Name": "a", "Count": "not-a-number"},
			matches:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			matcher := DecodesTo(testCase.expected)

			require.Equal(t, testCase.matches, matcher(testCase.actual))
		})
	}
}

func TestEmptyToolContext(t *testing.T) {
	t.Run("returns zero values with initialized maps", func(t *testing.T) {
		toolContext := EmptyToolContext()

		require.NotNil(t, toolContext.Params)
		require.Empty(t, toolContext.Params)
		require.Nil(t, toolContext.Workload)
		require.Empty(t, toolContext.WorkingDir)
		require.NotNil(t, toolContext.Env)
		require.Empty(t, toolContext.Env)
		require.Zero(t, toolContext.Timeout)
		require.Empty(t, toolContext.ToolsRoot)
		require.NotNil(t, toolContext.Metadata)
		require.Empty(t, toolContext.Metadata)
	})
}
