# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Replaces core/Taskfile.yml.

performix_discover_go_modules(PERFORMIX_GO_MODULES "${PERFORMIX_CORE_DIR}")
if(NOT PERFORMIX_GO_MODULES)
  message(FATAL_ERROR "No Go modules found under ${PERFORMIX_CORE_DIR}")
endif()

message(STATUS "Go modules:")
foreach(_module IN LISTS PERFORMIX_GO_MODULES)
  file(RELATIVE_PATH _rel "${PERFORMIX_ROOT_DIR}" "${_module}")
  message(STATUS "  ${_rel}")
endforeach()

# ---------------------------------------------------------------------------
# Generated message codes
#
# core/apap-engine/message/codes.go is derived from the English message catalogue
# and is deliberately not checked in, so it has to exist before anything that
# imports the message package will compile. The shell script version regenerated
# it unconditionally; this rule only reruns the generator when its inputs change.
# ---------------------------------------------------------------------------

set(PERFORMIX_MESSAGE_CODES "${PERFORMIX_CORE_DIR}/apap-engine/message/codes.go")

add_custom_command(
  OUTPUT "${PERFORMIX_MESSAGE_CODES}"
  COMMAND "${CMAKE_COMMAND}" -E env ${PERFORMIX_GO_BASE_ENV}
          "${PERFORMIX_GO_EXECUTABLE}" run gen_codes.go
  WORKING_DIRECTORY "${PERFORMIX_CORE_DIR}/apap-engine/message-gen"
  DEPENDS
    "${PERFORMIX_CORE_DIR}/apap-engine/message-gen/gen_codes.go"
    "${PERFORMIX_CORE_DIR}/apap-engine/message/catalog_en-US.json"
  COMMENT "Generating engine message codes"
  VERBATIM)

add_custom_target(message-codes DEPENDS "${PERFORMIX_MESSAGE_CODES}")

# ---------------------------------------------------------------------------
# Optional gRPC regeneration
#
# The generated stubs under core/clients/go are checked in, so protoc is only
# needed when the .proto files change. Keeping this opt-in means the ordinary
# build does not require a protobuf compiler at all.
# ---------------------------------------------------------------------------

if(PERFORMIX_REGENERATE_PROTO)
  include(PerformixProto)
endif()

# ---------------------------------------------------------------------------
# Components
# ---------------------------------------------------------------------------

add_subdirectory(apap-cli)

if(PERFORMIX_BUILD_AGENT)
  add_subdirectory(atperf-agent)
endif()

# ---------------------------------------------------------------------------
# Developer workflow targets
# ---------------------------------------------------------------------------

# Populate the module cache. Useful as a single fetch step before an otherwise
# network-isolated build, and as the point at which a controlled module proxy is
# applied through PERFORMIX_GOPROXY.
performix_go_command(deps
  ARGS mod download
  COMMENT "Downloading Go module dependencies")

# The generators write into the source tree, which is why this is not part of
# the default build. message-codes above covers the one generated file that the
# build actually depends on.
performix_go_command(generate
  ARGS generate ./...
  COMMENT "Running go generate across all modules"
  DEPENDS message-codes)

find_program(PERFORMIX_GOLANGCI_LINT NAMES golangci-lint)
if(PERFORMIX_GOLANGCI_LINT)
  set(_lint_commands)
  foreach(_module IN LISTS PERFORMIX_GO_MODULES)
    list(APPEND _lint_commands
      COMMAND "${CMAKE_COMMAND}" -E chdir "${_module}"
              "${CMAKE_COMMAND}" -E env ${PERFORMIX_GO_BASE_ENV}
              "${PERFORMIX_GOLANGCI_LINT}" run
              "--build-tags=${PERFORMIX_GO_BUILD_TAGS}"
              ./...
              --timeout 300s
              --max-issues-per-linter 0
              --max-same-issues 0)
  endforeach()
  add_custom_target(lint
    ${_lint_commands}
    COMMENT "Linting all Go modules"
    VERBATIM)
  add_dependencies(lint message-codes)
else()
  add_custom_target(lint
    COMMAND "${CMAKE_COMMAND}" -E echo
            "golangci-lint not found; install it and re-run cmake to enable this target"
    COMMAND "${CMAKE_COMMAND}" -E false
    VERBATIM)
endif()

# ---------------------------------------------------------------------------
# Tests
#
# One CTest entry per Go module, so "ctest -R apap-engine" and friends work.
# "cmake --build . --target check" builds the generated sources first and then
# runs the suite.
# ---------------------------------------------------------------------------

foreach(_module IN LISTS PERFORMIX_GO_MODULES)
  file(RELATIVE_PATH _rel "${PERFORMIX_CORE_DIR}" "${_module}")
  add_test(
    NAME "go:${_rel}"
    COMMAND "${PERFORMIX_GO_EXECUTABLE}" test
            "-tags=${PERFORMIX_GO_BUILD_TAGS}" ./...
    WORKING_DIRECTORY "${_module}")
  set_tests_properties("go:${_rel}" PROPERTIES
    ENVIRONMENT "GOTOOLCHAIN=${PERFORMIX_GOTOOLCHAIN};GOFLAGS=${PERFORMIX_GOFLAGS}"
    TIMEOUT 1800)
endforeach()

add_custom_target(check
  COMMAND "${CMAKE_CTEST_COMMAND}" --output-on-failure
  WORKING_DIRECTORY "${CMAKE_BINARY_DIR}"
  COMMENT "Running Go unit tests"
  VERBATIM)
add_dependencies(check message-codes)
