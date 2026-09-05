# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# The apx CLI. Replaces the build:apx task and core/apap-cli/makefile.

if(PERFORMIX_BUILD_CLI)

  # apap-cli replaces the sibling modules through go.mod replace directives, so
  # a change anywhere in the workspace has to invalidate the binary.
  set(_apx_watch
    "${CMAKE_CURRENT_SOURCE_DIR}"
    "${PERFORMIX_CORE_DIR}/apap-engine"
    "${PERFORMIX_CORE_DIR}/atperf-agent"
    "${PERFORMIX_CORE_DIR}/atperf-compatibility"
    "${PERFORMIX_CORE_DIR}/atperf-version"
    "${PERFORMIX_CORE_DIR}/clients/go")

  set(_apx_ldflags "")
  if(PERFORMIX_INJECT_VERSION)
    set(_apx_ldflags
      "-X github.com/Arm-Debug/apap-cli/atperf-version/versions.version=${PERFORMIX_ENGINE_VERSION}")
  endif()

  performix_go_binary(
    TARGET apx
    OUTPUT "${PERFORMIX_APX_BINARY}"
    MODULE_DIR "${CMAKE_CURRENT_SOURCE_DIR}"
    PACKAGE "."
    CGO ${PERFORMIX_ENABLE_CGO}
    TAGS ${PERFORMIX_GO_BUILD_TAGS}
    LDFLAGS "${_apx_ldflags}"
    WATCH ${_apx_watch}
    COMMENT "Building apx")

  # A custom command OUTPUT is only visible to targets in the directory that
  # declared it, so the generated message codes are pulled in as a target-level
  # dependency rather than a file-level one.
  add_dependencies(apx message-codes)

endif()

if(PERFORMIX_BUILD_TOOL_BUNDLES)
  add_subdirectory(tools-builtin)
endif()
