<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# JavaScript Unit Tests

The `jstest` package provides Go harnesses for testing JavaScript with Goja. A test typically loads a JavaScript file, configures any Go mocks used by the JavaScript, calls a function, and asserts on its result or error.

Use `LoadJSModule` for functions exported from a CommonJS module, or `LoadJSScript` for functions defined in the global scope.

```go
harness := jstest.LoadJSModule(t, "tool-integrations/utils.js")

engine := MockEngine{}
engine.On(
	"ExecCommand",
	mock.MatchedBy(jstest.DecodesTo([]string{"logname"})),
	&tool.ExecOptions{},
).Return(tool.CommandResult{RC: 0, Stdout: "test-user"}, nil)

var result string
require.NoError(t, harness.CallWithDest(t, "resolveLoginName", &result, &engine))
assert.Equal(t, "test-user", result)
engine.AssertExpectations(t)
```

## Type Conversion

Goja converts values as they cross the Go and JavaScript boundary. Harness call arguments can be Go values or `goja.Value` values. Parameters of mocked Go methods can accept specified Go types (like a corresponding struct), `any` (for the js arg to be exported to its natural Go representation), or `goja.Value` (to handle conversion manually). `DecodesTo` is useful for matching structured values such as JavaScript arrays and objects for methods which accept a more generic type, like `any`.

`Call` exports the result to its natural Go representation. For example, arrays become `[]any` and objects become `map[string]any`. Use `CallWithDest` with a pointer to export the result into a specific Go type.

Any exceptions thrown in the JavaScript function are returned as errors. Exceptions which follow the CatalogMessage structure (see [jsdocs.js:CatalogMessage](../../apap-cli/recipes/docs/jsdocs.js)) are converted into Go `message.Message`s; other errors are returned as plain `gojautils.ScriptError`s.
