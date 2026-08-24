// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package tool_goja

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dop251/goja/parser"
)

// JsonFieldNameMapper implements goja.FieldNameMapper - it maps the fields
// and methods on Go structs to their JS equivalents as follows:
//   - If LowercaseMethodNames is true, method names have their first
//     letter made lowercase
//     e.g. "ExecCommand" -> "execCommand"
//   - Otherwise, method names are left unchanged
//   - Field names are mapped to their `json` tags if they exist;
//     otherwise, they stay the same
//     e.g. "MyField" -> "MyField"
//     "MyField `json:\"something_else\"` -> "something_else"
type JsonFieldNameMapper struct {
	LowercaseMethodNames bool
}

func (j *JsonFieldNameMapper) FieldName(
	_ reflect.Type,
	field reflect.StructField,
) string {
	tag := strings.Split(field.Tag.Get("json"), ",")[0]

	switch tag {
	case "-":
		return ""
	case "":
		return field.Name
	default:
		if parser.IsIdentifier(tag) {
			return tag
		}
		return ""
	}
}

func (j *JsonFieldNameMapper) MethodName(
	_ reflect.Type,
	method reflect.Method,
) string {
	if j.LowercaseMethodNames {
		return lowerFirstRune(method.Name)
	}
	return method.Name
}

func lowerFirstRune(value string) string {
	if value == "" {
		return ""
	}

	first, size := utf8.DecodeRuneInString(value)
	return string(unicode.ToLower(first)) + value[size:]
}
