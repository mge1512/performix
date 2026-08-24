// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package tool_goja

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFieldName(t *testing.T) {
	type fields struct {
		Plain   string
		Custom  string `json:"customName,omitempty"`
		Empty   string `json:",omitempty"`
		Ignored string `json:"-"`
	}

	testCases := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{
			name:      "uses Go field name without JSON tag",
			fieldName: "Plain",
			expected:  "Plain",
		},
		{
			name:      "uses JSON tag and ignores options",
			fieldName: "Custom",
			expected:  "customName",
		},
		{
			name:      "uses Go field name for empty JSON tag",
			fieldName: "Empty",
			expected:  "Empty",
		},
		{
			name:      "omits ignored JSON field",
			fieldName: "Ignored",
			expected:  "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			structType := reflect.TypeOf(fields{})
			field, found := structType.FieldByName(testCase.fieldName)
			require.True(t, found)

			mapper := JsonFieldNameMapper{}
			require.Equal(t, testCase.expected, mapper.FieldName(structType, field))
		})
	}
}

func TestMethodName(t *testing.T) {
	t.Run("lowercases first Unicode rune if LowercaseMethodNames is true", func(t *testing.T) {
		mapper := JsonFieldNameMapper{LowercaseMethodNames: true}
		method := reflect.Method{Name: "MyMethod"}

		require.Equal(t, "myMethod", mapper.MethodName(nil, method))
	})
	t.Run("leaves names untouches if LowercaseMethodNames is false", func(t *testing.T) {
		mapper := JsonFieldNameMapper{}
		method := reflect.Method{Name: "MyMethod"}

		require.Equal(t, "MyMethod", mapper.MethodName(nil, method))
	})
}
