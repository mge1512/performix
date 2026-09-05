<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Building Arm Performix with CMake

This is an alternative to the `bootstrap` + `mise` + `task` route described in
the README. It builds the same artefacts from the same sources. The two can
coexist; nothing in the Taskfiles was removed.

## Why

`bootstrap` downloads a pinned `mise` binary from GitHub Releases, checks it
against an in-tree SHA-256, writes a block into the user's shell profile, and
then has `mise` fetch four more pinned tools. That gets a developer to a working
build quickly, but it also means the toolchain arrives outside whatever package
manager and signing chain the build host already trusts, and it does not work on
a host without outbound access to release hosting.

CMake takes the other approach. The pinned versions become *minimum* versions
that are checked against what the host already provides, and configure fails with
an actionable message if the host is short. The toolchain stays under the control
of the platform, which is what a controlled build service needs.

## Prerequisites

| Tool | Version | Needed for |
| --- | --- | --- |
| CMake | 3.20 or later | always |
| Go | 1.26.6 or later | always |
| C compiler | any working toolchain | the CLI, which links duckdb through cgo |
| GNU tar | any | optional, improves archive reproducibility |
| `protoc` | 23.4 or later | only with `-DPERFORMIX_REGENERATE_PROTO=ON` |
| `golangci-lint` | 2.x | only for the `lint` target |

On SLE or openSUSE:

```bash
zypper install cmake go1.26 gcc tar
```

Python is not required. The build path that previously ran `get-tools.py`,
`build_target_agent.py`, `bundle_tool.py`, `get_atperf_version.py` and
`terminology.py` is expressed directly in CMake.

## Build

```bash
cmake -S . -B build
cmake --build build -j"$(nproc)"
```

That is the equivalent of `mise exec -- task install`. The result is:

```
build/bin/apx
build/bin/tools/apx-agent/1.21.0/apx-agent-<Os>-<Arch>.tar.gz
build/bin/tools/parquet-to-json/1.21.0/parquet-to-json-<Os>-<Arch>.tar.gz
build/bin/tools/sysutil-timeline/1.21.0/sysutil-timeline-<Os>-<Arch>.tar.gz
```

`apx` resolves its bundles relative to its own executable, so the binary and the
`tools/` tree have to stay siblings. To reproduce the historical in-tree layout
that the README examples and the Robot suite assume:

```bash
cmake -S . -B build -DPERFORMIX_STAGE_DIR="$PWD/core/apap-cli"
```

## Targets

| Target | Replaces |
| --- | --- |
| default (`all`) | `task install` |
| `apx` | `task core:build:apx` |
| `tools` | `task core:deps:tools` |
| `agent` | `task core:build:agent` |
| `deps` | fetching Go modules, previously implicit |
| `generate` | `task core:generate`, Go half |
| `proto` | `scripts/protobuf/generate.sh`, opt-in |
| `lint` | `scripts/lint-all.sh`, Go half |
| `check` | `scripts/go/test-all.sh` |
| `install` | no previous equivalent |

Tests are also registered with CTest, one entry per Go module:

```bash
ctest --test-dir build --output-on-failure
ctest --test-dir build -R apap-engine
```

## Options

| Option | Default | Meaning |
| --- | --- | --- |
| `PERFORMIX_BUILD_CLI` | `ON` | Build `apx` |
| `PERFORMIX_BUILD_TOOL_BUNDLES` | `ON` | Build the built-in tool bundles |
| `PERFORMIX_BUILD_AGENT` | `ON` | Build the target agent bundles |
| `PERFORMIX_ENABLE_CGO` | `ON` | Required by duckdb; turning it off drops the C compiler requirement |
| `PERFORMIX_SNAPSHOT` | `OFF` | Append `-dev` to the bundle version |
| `PERFORMIX_INJECT_VERSION` | `OFF` | Inject the version into `apx` via `-ldflags -X` |
| `PERFORMIX_REGENERATE_PROTO` | `OFF` | Regenerate the gRPC stubs |
| `PERFORMIX_REPRODUCIBLE_ARCHIVES` | `ON` | Pin timestamps and ownership in bundles |
| `PERFORMIX_GO_BUILD_TAGS` | `duckdb_arrow` | Tags for build, test and lint |
| `PERFORMIX_STAGE_DIR` | `<build>/bin` | Where `apx` and `tools/` are placed |
| `PERFORMIX_GO_MINIMUM_VERSION` | `1.26.6` | Version gate |
| `PERFORMIX_GOTOOLCHAIN` | `local` | Value of `GOTOOLCHAIN` |
| `PERFORMIX_GOFLAGS` | `-mod=readonly` | Value of `GOFLAGS` |
| `PERFORMIX_GOPROXY` | empty | Module proxy; empty inherits the environment |
| `PERFORMIX_GOSUMDB` | empty | Checksum database |
| `PERFORMIX_ENGINE_VERSION` | from `version.py` | Bundle version override |

## Notes on the design

**`GOTOOLCHAIN=local` is the default.** With Go's own default of `auto`, the
toolchain reads the `go 1.26.6` directive from `go.mod` and downloads a matching
toolchain if the installed one is older. That would silently defeat both the
version check above and the point of installing Go from a package. Setting
`local` makes the installed toolchain the one that is used, and makes a mismatch
an error rather than a download.

**Module fetching is a separate step.** `cmake --build build --target deps` runs
`go mod download` in every module. Run it once against a controlled module proxy
via `-DPERFORMIX_GOPROXY=...`, then build with no further network access.
`GOFLAGS=-mod=readonly` keeps the build from editing `go.mod` behind your back.

**Bundles are reproducible.** Entries are sorted, ownership is pinned to 0:0 and
timestamps to `SOURCE_DATE_EPOCH`, taken from the environment, then from the last
git commit, then from a fixed fallback. Two builds of the same tree produce
byte-identical archives. GNU tar is used when present because it can pin
ownership; `cmake -E tar` is the fallback and pins timestamps only.

**Generated sources.** `core/apap-engine/message/codes.go` is generated from
`catalog_en-US.json` and is a hard prerequisite for everything that imports the
message package. The rule only reruns when its inputs change, rather than on
every build. The first build after it appears triggers one CMake reconfigure,
because the file lands inside a globbed source tree; this is a one-off.

The gRPC stubs under `core/clients/go` are checked in, so `protoc` is not needed
for an ordinary build. `PERFORMIX_REGENERATE_PROTO=ON` enables the `proto` and
`proto-mocks` targets. Unlike the shell script, these locate the plugins on
`PATH` rather than running `go install` as a side effect of code generation.

**Installation.** `apx` finds its bundles next to its own executable, so both are
installed under `libexec/performix` with a symlink in `bindir`. Go resolves the
symlink through `/proc/self/exe`, so the lookup still lands correctly. `DESTDIR`
is honoured:

```bash
DESTDIR=/tmp/stage cmake --install build --prefix /usr
```

## Not covered

- `prepare_release_tool_dirs`, that is the `core/apap-cli/tools-<goos>-<goarch>`
  staging trees. They are consumed by release packaging rather than by the
  developer build; release still runs the existing scripts.
- Robot Framework tests, the Python and JavaScript lint paths, the AI-insights
  evaluation harness, and code signing. None of these were on the
  `task install` path.
- The `sysutil-timeline` bundle contains a `LICENSE` symlink pointing outside its
  own tree, so it does not resolve after extraction. This matches the behaviour
  of the previous `tarfile`-based packaging rather than fixing it.
