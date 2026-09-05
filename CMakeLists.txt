# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Top-level build orchestration for Arm Performix.
#
# This replaces the mise + go-task pair:
#
#   mise.toml / bootstrap  -> configure-time toolchain discovery and version gate
#   Taskfile.yml           -> the custom targets defined below
#   core/Taskfile.yml      -> core/CMakeLists.txt
#
# The difference in approach is deliberate. mise downloads pinned tool binaries
# from GitHub Releases at bootstrap time. CMake instead requires the tools to be
# present already, checks their versions, and fails with an actionable message if
# they are too old. That keeps the toolchain inside whatever package manager and
# signing chain the build host already trusts, and makes the build usable on
# machines with no outbound network access to release hosting.
#
# Usage:
#   cmake -S . -B build
#   cmake --build build -j"$(nproc)"
#
# Named targets (cmake --build build --target <name>):
#   apx            Build the Performix CLI only
#   tools          Build the built-in tool bundles
#   agent          Build the target agent bundles
#   generate       Run "go generate ./..." across every Go module
#   deps           Populate the Go module cache ("go mod download")
#   lint           Run golangci-lint across every Go module
#   check          Run the Go unit tests through CTest
#   proto          Regenerate the gRPC stubs (requires -DPERFORMIX_REGENERATE_PROTO=ON)

cmake_minimum_required(VERSION 3.20)

list(APPEND CMAKE_MODULE_PATH "${CMAKE_CURRENT_LIST_DIR}/cmake")

set(PERFORMIX_ROOT_DIR "${CMAKE_CURRENT_LIST_DIR}")
set(PERFORMIX_CORE_DIR "${PERFORMIX_ROOT_DIR}/core")
set(PERFORMIX_CMAKE_SCRIPT_DIR "${PERFORMIX_ROOT_DIR}/cmake/scripts")

# ---------------------------------------------------------------------------
# Project version
#
# core/atperf-version/versions/version.py is the single source of truth. It is
# embedded into the Go binaries and previously read back out through
# "go run ." in scripts/get_atperf_version.py. Reading the file directly removes
# a Go and a Python invocation from the configure step.
# ---------------------------------------------------------------------------

set(PERFORMIX_VERSION_FILE "${PERFORMIX_CORE_DIR}/atperf-version/versions/version.py")
if(NOT EXISTS "${PERFORMIX_VERSION_FILE}")
  message(FATAL_ERROR "Version file not found: ${PERFORMIX_VERSION_FILE}")
endif()
file(STRINGS "${PERFORMIX_VERSION_FILE}" _performix_version_line REGEX "^[ \t]*VERSION[ \t]*=")
if(NOT _performix_version_line MATCHES "\"([^\"]+)\"")
  message(FATAL_ERROR "Could not parse VERSION from ${PERFORMIX_VERSION_FILE}")
endif()
set(PERFORMIX_BASE_VERSION "${CMAKE_MATCH_1}")

project(Performix
  VERSION "${PERFORMIX_BASE_VERSION}"
  DESCRIPTION "Arm Performix performance analysis toolkit"
  HOMEPAGE_URL "https://github.com/arm/performix"
  LANGUAGES NONE)

# ---------------------------------------------------------------------------
# Options
# ---------------------------------------------------------------------------

option(PERFORMIX_BUILD_CLI          "Build the apx CLI"                                ON)
option(PERFORMIX_BUILD_TOOL_BUNDLES "Build the built-in tool bundles"                  ON)
option(PERFORMIX_BUILD_AGENT        "Build the target agent bundles"                   ON)
option(PERFORMIX_ENABLE_CGO         "Build the CLI with cgo (required by duckdb)"      ON)
option(PERFORMIX_SNAPSHOT           "Mark bundles as a snapshot build (-dev suffix)"   OFF)
option(PERFORMIX_INJECT_VERSION     "Inject the version into the CLI via -ldflags -X"  OFF)
option(PERFORMIX_REGENERATE_PROTO   "Regenerate the gRPC stubs from core/api"          OFF)
option(PERFORMIX_REPRODUCIBLE_ARCHIVES
                                    "Normalise timestamps and ownership in bundles"    ON)

set(PERFORMIX_GO_BUILD_TAGS "duckdb_arrow" CACHE STRING
    "Build tags passed to go build, go test and golangci-lint")

set(PERFORMIX_STAGE_DIR "${CMAKE_BINARY_DIR}/bin" CACHE PATH
    "Directory holding the built apx binary and its adjacent tools/ tree")

set(PERFORMIX_ENGINE_VERSION "" CACHE STRING
    "Override the engine version used for bundle directory names")

if(NOT PERFORMIX_ENGINE_VERSION)
  set(PERFORMIX_ENGINE_VERSION "${PERFORMIX_BASE_VERSION}")
  if(PERFORMIX_SNAPSHOT)
    string(APPEND PERFORMIX_ENGINE_VERSION "-dev")
  endif()
endif()

set(PERFORMIX_TOOLS_DIR "${PERFORMIX_STAGE_DIR}/tools")

# apx resolves its tool bundles relative to its own executable, so the binary and
# the tools/ tree have to stay siblings. Set PERFORMIX_STAGE_DIR to
# <source>/core/apap-cli to reproduce the historical in-tree layout that the
# README examples and the Robot suite assume.
file(MAKE_DIRECTORY "${PERFORMIX_STAGE_DIR}")

if(PERFORMIX_ENABLE_CGO)
  # duckdb-go-bindings links prebuilt static libraries through cgo, so a working
  # C toolchain is a hard requirement for the CLI. Let CMake verify it here
  # rather than letting go build fail later with a less obvious message.
  enable_language(C)
endif()

include(GNUInstallDirs)
include(PerformixGo)
include(PerformixBundle)

set(PERFORMIX_APX_BINARY "${PERFORMIX_STAGE_DIR}/apx${PERFORMIX_GO_EXE}")

# ---------------------------------------------------------------------------
# Reproducible archive timestamps
# ---------------------------------------------------------------------------

set(PERFORMIX_SOURCE_DATE_EPOCH "$ENV{SOURCE_DATE_EPOCH}" CACHE STRING
    "Fixed timestamp for bundle entries; taken from the environment by default")

if(PERFORMIX_REPRODUCIBLE_ARCHIVES AND NOT PERFORMIX_SOURCE_DATE_EPOCH)
  find_package(Git QUIET)
  if(GIT_EXECUTABLE AND EXISTS "${PERFORMIX_ROOT_DIR}/.git")
    execute_process(
      COMMAND "${GIT_EXECUTABLE}" log -1 --pretty=%ct
      WORKING_DIRECTORY "${PERFORMIX_ROOT_DIR}"
      OUTPUT_VARIABLE PERFORMIX_SOURCE_DATE_EPOCH
      OUTPUT_STRIP_TRAILING_WHITESPACE
      ERROR_QUIET)
  endif()
  if(NOT PERFORMIX_SOURCE_DATE_EPOCH)
    # Falling back to a fixed value keeps repeated builds of the same tree
    # byte-identical even without a git checkout.
    set(PERFORMIX_SOURCE_DATE_EPOCH "1767225600")
  endif()
endif()

# ---------------------------------------------------------------------------
# Subdirectories
# ---------------------------------------------------------------------------

enable_testing()

add_subdirectory(core)

# ---------------------------------------------------------------------------
# Aggregate targets
# ---------------------------------------------------------------------------

add_custom_target(performix ALL)
if(TARGET apx)
  add_dependencies(performix apx)
endif()
if(TARGET tools)
  add_dependencies(performix tools)
endif()
if(TARGET agent)
  add_dependencies(performix agent)
endif()

# ---------------------------------------------------------------------------
# Installation
#
# apx locates its bundles next to the executable, so the binary and tools/ are
# installed together under a private libexec directory and exposed through a
# symlink in bindir. Go resolves the symlink through /proc/self/exe, so the
# bundle lookup still lands in the private directory.
# ---------------------------------------------------------------------------

set(PERFORMIX_INSTALL_LIBEXECDIR "${CMAKE_INSTALL_LIBEXECDIR}/performix")

if(PERFORMIX_BUILD_CLI)
  install(PROGRAMS "${PERFORMIX_APX_BINARY}"
          DESTINATION "${PERFORMIX_INSTALL_LIBEXECDIR}")
  install(CODE "
    file(MAKE_DIRECTORY \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${CMAKE_INSTALL_BINDIR}\")
    file(CREATE_LINK
      \"\${CMAKE_INSTALL_PREFIX}/${PERFORMIX_INSTALL_LIBEXECDIR}/apx\"
      \"\$ENV{DESTDIR}\${CMAKE_INSTALL_PREFIX}/${CMAKE_INSTALL_BINDIR}/apx\"
      SYMBOLIC)
  ")
endif()

if(PERFORMIX_BUILD_TOOL_BUNDLES OR PERFORMIX_BUILD_AGENT)
  install(DIRECTORY "${PERFORMIX_TOOLS_DIR}/"
          DESTINATION "${PERFORMIX_INSTALL_LIBEXECDIR}/tools"
          FILES_MATCHING PATTERN "*.tar.gz")
endif()

# ---------------------------------------------------------------------------
# Configuration summary
# ---------------------------------------------------------------------------

message(STATUS "")
message(STATUS "Performix ${PERFORMIX_ENGINE_VERSION}")
message(STATUS "  Go toolchain     : ${PERFORMIX_GO_EXECUTABLE} (${PERFORMIX_GO_VERSION})")
message(STATUS "  Host platform    : ${PERFORMIX_GO_HOSTOS}/${PERFORMIX_GO_HOSTARCH}")
message(STATUS "  Build tags       : ${PERFORMIX_GO_BUILD_TAGS}")
message(STATUS "  cgo              : ${PERFORMIX_ENABLE_CGO}")
message(STATUS "  Stage directory  : ${PERFORMIX_STAGE_DIR}")
message(STATUS "  CLI              : ${PERFORMIX_BUILD_CLI}")
message(STATUS "  Tool bundles     : ${PERFORMIX_BUILD_TOOL_BUNDLES}")
message(STATUS "  Agent bundles    : ${PERFORMIX_BUILD_AGENT}")
message(STATUS "  Proto codegen    : ${PERFORMIX_REGENERATE_PROTO}")
message(STATUS "")
