# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Built-in tool bundles. Replaces the public tool half of
# core/scripts/get-tools.py, that is the deps:tools task.

set(_bundles)

# ---------------------------------------------------------------------------
# parquet-to-json
#
# Lives inside the apap-cli module rather than having its own go.mod, so go
# build runs from the tool directory and resolves through the parent module.
# ---------------------------------------------------------------------------

set(_parquet_dir "${CMAKE_CURRENT_SOURCE_DIR}/parquet-to-json")

# goos|goarch|OsLabel|ArchLabel
set(_parquet_targets
  "linux|arm64|Linux|aarch64"
  "linux|amd64|Linux|x86_64"
  "windows|arm64|Windows|aarch64"
  "windows|amd64|Windows|x86_64"
  "darwin|arm64|Darwin|aarch64"
  "darwin|amd64|Darwin|x86_64")

foreach(_target IN LISTS _parquet_targets)
  string(REPLACE "|" ";" _fields "${_target}")
  list(GET _fields 0 _goos)
  list(GET _fields 1 _goarch)
  list(GET _fields 2 _os_label)
  list(GET _fields 3 _arch_label)

  performix_go_tool_bundle(
    TOOL_NAME parquet-to-json
    MODULE_DIR "${_parquet_dir}"
    PACKAGE "."
    GOOS "${_goos}"
    GOARCH "${_goarch}"
    OS_LABEL "${_os_label}"
    ARCH_LABEL "${_arch_label}"
    WATCH "${_parquet_dir}" "${PERFORMIX_CORE_DIR}/apap-cli/go.mod"
    ARCHIVE_VAR _archive)
  list(APPEND _bundles "${_archive}")
endforeach()

# ---------------------------------------------------------------------------
# sysutil-timeline
#
# Shipped as sources rather than as a compiled binary. The exclusions match the
# filter that the Python packaging step applied.
# ---------------------------------------------------------------------------

# OsLabel|ArchLabel
set(_sysutil_targets
  "Linux|aarch64"
  "Linux|x86_64")

foreach(_target IN LISTS _sysutil_targets)
  string(REPLACE "|" ";" _fields "${_target}")
  list(GET _fields 0 _os_label)
  list(GET _fields 1 _arch_label)

  performix_payload_bundle(
    TOOL_NAME sysutil-timeline
    SOURCE_DIR "${CMAKE_CURRENT_SOURCE_DIR}/sysutil-timeline"
    OS_LABEL "${_os_label}"
    ARCH_LABEL "${_arch_label}"
    EXCLUDE "(^|/)__pycache__(/|$)|(^|/)\\.pytest_cache(/|$)|(^|/)env(/|$)|(^|/)tests(/|$)|\\.pyc$"
    ARCHIVE_VAR _archive)
  list(APPEND _bundles "${_archive}")
endforeach()

add_custom_target(tools DEPENDS ${_bundles})
