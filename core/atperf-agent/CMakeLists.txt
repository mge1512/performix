# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Target agent bundles. Replaces the build:agent task, which called
# get-tools.py, which in turn drove GoReleaser through .goreleaser-agent.yml.
#
# The GoReleaser configuration reduced to: build apx-agent for seven targets
# with CGO_ENABLED=0 and -trimpath, strip symbols, inject the version, and pack
# each binary together with its launcher script. That is expressed directly
# below, which removes GoReleaser and the Python wrapper from the build path.

# The binary name is defined once in the terminology file and read from it here
# so the two cannot drift apart.
file(READ "${PERFORMIX_CORE_DIR}/apap-engine/terminology/terminology.json" _terminology)
string(JSON PERFORMIX_AGENT_BINARY GET "${_terminology}" "AGENT_BINARY_NAME")

set(_agent_dir "${CMAKE_CURRENT_SOURCE_DIR}")
set(_agent_scripts "${PERFORMIX_CORE_DIR}/scripts")

set(_agent_watch
  "${_agent_dir}"
  "${PERFORMIX_CORE_DIR}/apap-engine"
  "${PERFORMIX_CORE_DIR}/atperf-version"
  "${PERFORMIX_CORE_DIR}/clients/go")

set(_agent_ldflags
  "-s -w -X github.com/Arm-Debug/apap-cli/atperf-version/versions.version=${PERFORMIX_ENGINE_VERSION}")

# goos|goarch|OsLabel|ArchLabel|launcher-source|launcher-name-in-archive
set(_agent_targets
  "android|arm64|Android|aarch64|verify-and-launch-android.sh|verify-and-launch.sh"
  "linux|arm64|Linux|aarch64|verify-and-launch.sh|verify-and-launch.sh"
  "linux|amd64|Linux|x86_64|verify-and-launch.sh|verify-and-launch.sh"
  "windows|arm64|Windows|aarch64|verify-and-launch.ps1|verify-and-launch.ps1"
  "windows|amd64|Windows|x86_64|verify-and-launch.ps1|verify-and-launch.ps1"
  "darwin|arm64|Darwin|aarch64|verify-and-launch.sh|verify-and-launch.sh"
  "darwin|amd64|Darwin|x86_64|verify-and-launch.sh|verify-and-launch.sh")

set(_agent_bundles)

foreach(_target IN LISTS _agent_targets)
  string(REPLACE "|" ";" _fields "${_target}")
  list(GET _fields 0 _goos)
  list(GET _fields 1 _goarch)
  list(GET _fields 2 _os_label)
  list(GET _fields 3 _arch_label)
  list(GET _fields 4 _launcher_src)
  list(GET _fields 5 _launcher_name)

  performix_go_tool_bundle(
    TOOL_NAME "${PERFORMIX_AGENT_BINARY}"
    MODULE_DIR "${_agent_dir}"
    PACKAGE "."
    GOOS "${_goos}"
    GOARCH "${_goarch}"
    OS_LABEL "${_os_label}"
    ARCH_LABEL "${_arch_label}"
    LDFLAGS "${_agent_ldflags}"
    EXTRA_FILE "${_agent_scripts}/${_launcher_src}"
    EXTRA_FILE_NAME "${_launcher_name}"
    WATCH ${_agent_watch}
    ARCHIVE_VAR _archive)
  list(APPEND _agent_bundles "${_archive}")
endforeach()

add_custom_target(agent DEPENDS ${_agent_bundles})

# The agent imports apap-engine/message, so the generated codes.go has to exist
# first. A custom command OUTPUT is only visible inside the directory that
# declared it, hence the target-level dependency.
add_dependencies(agent message-codes)
