<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Performix Recipe JS Testing

## Introduction

Legacy link: https://confluence.arm.com/spaces/ITS/pages/2910958088/Performix+Recipe+JS+Testing

Performix recipes and tool integrations are shipped as JavaScript and execute inside the APX Engine under the Goja runtime. Shipped JS is already exercised by a mixture of:

- Robot tests
- Recipe parser and stage tests
- Goja tool-integration tests
- Renderer focused tests

However, this coverage is distributed across several suites and does not provide a consistent, reusable approach for focused JS behaviour and JS-to-Go boundary testing without a real target involved.

This record compares Vitest under Node with Go tests that execute JavaScript under Goja. It selects a primary framework for recipes and tool integrations. Robot remains as a smaller end-to-end layer for behavior that genuinely requires a target or packaged product.

> This TDR selects the primary testing architecture for shipped recipes and tool-integrations. It describes the intended testing boundaries but does NOT define the final harness API, those will need to be resolved during implementation or in follow-up designs

## User-Level Features Enabled

The decision in this document does not directly provide any user-level features or benefits, but will improve the quality and delivery of Performix

- Better coverage of shipped JavaScript behavior and configuration.
- Faster development and review of recipe changes.
- Less dependence on target infrastructure for routine validation.

## Goals and Constraints

### Goals

- Establish the expected contributor workflow for JavaScript testing
- Support focused JavaScript behavior tests and engine-integration tests.
- Run deterministically without requiring a real target.
- Compare runtime fidelity, diagnostics, coverage, and harness maintenance.
- Make common recipe, stage, run-model and render fixtures concise and reusable so that fixture setup does not dominate individual tests.

### Constraints

- The shipped runtime is Goja, while Vitest runs under Node.
- Recipes depend on globals installed by Go and generally do not export helpers as Node modules.
- Tool integrations use asynchronous behavior, process handles, files, and Go-bound engine APIs.
- Tests must identify which production components execute and which are replaced by test doubles.

### Non-goals

- Replacing all Robot or end-to-end testing; some of these are still valuable.
- Building a framework for every JavaScript module in the repository.
- Defining every final harness API, fixture builder, or CI task in this record.

## Design

### Option 1: Vitest under Node

#### Writing and running tests

Tests are JavaScript or TypeScript files using Vitest's `describe`, `it`, `expect`, mocks, and spies. Contributors run them through npm scripts and can use filtering, watch mode, debugging, and V8 JavaScript coverage.

```shell
npm test
npm run test:coverage
```

#### Harness

Because shipped recipes are not Node modules, a Vitest harness must either load them through a Node VM or refactor pure helpers into importable modules. The VM approach installs JavaScript versions of Performix globals, exposes otherwise private helpers, and provides JavaScript substitutes for stage APIs, the engine, processes, files, promises, and timers.

Vitest provides the strongest JavaScript-native authoring experience. However, its harness duplicates contracts implemented in Go, and Node substitutes for the production runtime and bindings. Exercising Go renderers, DuckDB sessions, or target-agent boundaries would require an additional bridge into the engine.

### Option 2: Go tests executing JavaScript under Goja

#### Writing and running tests

Tests are Go `_test.go` files using `testing`, `testify`, and existing Go mocks. The shipped code under test remains JavaScript and executes in Goja; discovery, fixtures, mocks, and assertions are written in Go.

```shell
cd core/apap-engine
go test -tags=duckdb_arrow ./recipejstest ./tool/goja
go test -tags=duckdb_arrow ./recipejstest -run TestCodeHotspots
```

Taskfile entry points should be implemented afterwards to provide a better developer experience.

#### Harness

The recipe harness loads a shipped file through `RecipeParserJS`, installs production globals in a fresh Goja runtime, and provides focused operations

The final API is outside the scope of this document, but the routine Go/JS boundary should be centralised in the harness. Tests should be able to pass and receive ordinary Go values without interacting directly with `goja.Value`.

For Example:

- `Load` parses and validates a recipe.
- `Call` invokes a named helper and normalizes the returned value.
- Stage helpers enter through production stage wrappers and concrete APIs.

This approach exercises the shipped runtime and JavaScript-to-Go bindings in one process. Its main costs are more verbose Go assertions, no Vitest watch mode and no native JS coverage support.

Assertions for JS behaviour are written in Go, increasing verbosity and requiring contributors to cross a language boundary. The approach also lacks Vitest-style mocking of internal JS functions and modules. Tests should focus on testing through public interfaces, but where internal substitution is genuinely required, the test harness may need a limited JS mocking mechanism. This is unlikely to be needed in the first implementation.

> Code coverage for shipped recipes and tool integrations is an important capability. Goja does not provide native statement or branch coverage, but coverage may be technically feasible through source instrumentation using a tool such as [Istanbul](https://istanbul.js.org/). This approach has not yet been validated against the Performix runtime and may prove to be a significant technical challenge in its own right.
>
> The likely approach for this would be:
>
> - Instrument the Javascript using Istanbul and create a mirrored source tree to store preprocessed source
> - Execute the instrumented sources under Goja
> - Extract `globalThis.__coverage__` for each runtime
> - Merge and publish the results
>
> The initial estimate is approximately 3 story points for harness support, with medium confidence, and an additional 2 story points for CI integration.
>
> This work is lower priority than establishing the initial test harness. Coverage instrumentation should be investigated once the core harness workflow has been validated.

##### Fixture creation

Fixture creation should be a core harness capability rather than incidental test setup. Tests should describe the domain data and behaviour under test without being dominated by low-level setup. Fixture builders should provide sensible defaults and allow relevant fields to be overridden.

##### Wrapper Recipe Construction

Tests of exported utility modules should load and call those modules directly and not require a dummy recipe

Some integration tests will still require a complete recipe to exercise the production parser or stage wrappers. These recipes should be produced through a shared wrapper-recipe builder or template rather than assembled independently using formatted JS strings.

#### Test Boundary

Goja hosted tests execute the shipped JavaScript with the production recipe or tool loaded int a Goja runtime. These tests do not exercise the target-agent or external gRPC transport on a real target. That remains the responsibility of the Robot suite.

### Hybrid alternative

A hybrid could use Vitest for pure JavaScript helpers and Goja for stage, tool, and render integration. It would provide stronger helper-test ergonomics, but would also introduce two loaders, two test systems, placement rules, likely duplicate cases, and additional CI maintenance. It should be reconsidered only if Performix develops a substantial runtime-independent JavaScript module layer.

### Comparison by test situation

| Test type         | Vitest under Node                                                                                                                     | Go tests under Goja                                                                                                                 |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| Focused helper    | Write the test in JavaScript. A custom loader exposes the helper. JavaScript coverage is available.                                   | Write the test in Go. The harness loads the real recipe and calls the helper in Goja. JavaScript coverage is not available natively |
| Recipe stage      | Replace the stage API with a JavaScript mock and check the calls made by the recipe. Go bindings and value conversion are not tested. | Run the production stage wrapper and API. Use a Go mock to record the request produced by the recipe.                               |
| Tool lifecycle    | Run probe, run, stop, or cancel with a JavaScript mock of the engine. Async behavior comes from Node.                                 | Run the production Goja loader, event loop, and bound engine. Mock only the external engine or target-agent call.                   |
| DuckDB and render | Check the renderer configuration returned by JavaScript. A bridge is needed to run the Go renderer and DuckDB session.                | Create DuckDB fixtures and run the production render stage, renderer, session, and query in the same test.                          |
| Maintenance       | Maintain the custom loader, Performix globals, JavaScript mocks, and any bridge to Go.                                                | Maintain the Go harness and fixture builders while reusing production APIs and mocks.                                               |

Focused helper tests use a test-only way to call the helper. Goja stage, tool, and render tests use the same engine entry points as production.

### Developer examples

#### Focused helper

The same helper behavior can be expressed naturally in either framework.

Vitest:

```javascript
it('passes the selected sampling frequency to neoprof', () => {
  const { getToolsArg } = loadCodeHotspots();
  const workload = { type: 'systemWide' };

  expect(toPlain(getToolsArg('high', workload))).toEqual({
    tools: [{ name: 'neoprof', args: ['-r', 'high'] }],
    workload,
  });
});
```

Goja:

```go
func TestCodeHotspotsPassesSamplingFrequencyToNeoprof(t *testing.T) {
    harness := recipejstest.Load(t, recipePath)
    workload := map[string]any{"type": "systemWide"}

    result := harness.Call(t, "getToolsArg", "high", workload)

    require.Equal(t, map[string]any{
        "tools": []any{
            map[string]any{
                "name": "neoprof",
                "args": []any{"-r", "high"},
            },
        },
        "workload": workload,
    }, result)
}
```

#### Goja integration boundary

A stage test enters through the production wrapper and records the converted request at the execution-context boundary:

```go
cleanup := harness.ExecuteRunStage(t, 0, recordingContext)
if cleanup != nil {
    defer cleanup()
}

request := recordingContext.integrationCalls[0][0]
require.Equal(t, "neoprof", request.Name)
require.Equal(t, "high", request.Params["sampling_frequency"])
require.IsType(t, &tool.WorkloadSystemWide{}, request.Workload)
```

The same harness pattern applies to tool lifecycle and render tests: keep the production loader, runtime, and bindings in the path, and replace only the named external boundary.

#### Isolated utility methods

The exact harness API remains outside of the scope of this record. However, tests for exported utility methods should not configure a Goja runtime directly. Goja setup and Go/JS value conversion should be provided by the shared harness.

The intended contributor experience is approximately:

```go
func TestTimelinePivotQueryUsesViewportPlaceholders(t *testing.T) {
      harness := recipejstest.Load(
          t,
          recipeTestPath(t, "lib/timeline_config.js"),
      )

      result := harness.Call(t, "buildTimelinePivotQuery")
      query := result.(string)

      require.Contains(t, query, "FROM {table}")
      require.Contains(t, query, "x_start >= {rangeStart}")
      require.Contains(t, query, "x_start < {rangeEnd}")
  }
```

### Decision

Adopt Go tests executing shipped JS under Goja as the primary testing system for Performix recipes / tool integrations. Build a small reusable harness around the production recipe parser, tool loader, event looper and engine boundaries. Go mocks and fakes can be used for external effects such as gRPC communications w/ the target agent.

This choice accepts less concise JavaScript assertions and no JavaScript coverage report without additional investigation and development. In return, one framework can cover JS behavioural testing, production bindings, DuckDB fixtures and rendered outputs without a cross-language bridge.

## Delivery Plan

1. Create the initial recipe JS harness and validate it with focused module and stage tests. Include the shared test setup described above.
2. Expand the harness and unit test coverage.
3. Ensure the unit test coverage and tests are run in PRs.
4. Implement recipe JS integration tests, and make necessary adjustments to the harness and docs to support this.
5. Review existing Robot tests against the new testing boundary.
   1. Move focused behavior into new JS tests where equivalent coverage can be provided, while retaining Robot coverage for on-target testing
   2. A Robot test should only be removed when equivalent coverage exists at the same architectural boundary.
