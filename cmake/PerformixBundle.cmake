# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Tool bundle helpers.
#
# Replaces the packaging half of core/scripts/get-tools.py and the GoReleaser
# invocation behind it. A bundle is a tar.gz named
#
#   tools/<tool>/<version>/<tool>-<OsLabel>-<ArchLabel>.tar.gz
#
# holding the tool payload at the archive root. The naming and layout match what
# apx expects at runtime, so the two schemes are interchangeable.

include_guard(GLOBAL)

set(PERFORMIX_ARCHIVE_SCRIPT "${PERFORMIX_CMAKE_SCRIPT_DIR}/performix_archive.cmake")

# GNU tar can pin ownership and timestamps; libarchive through "cmake -E tar"
# can pin timestamps only. Prefer GNU tar when it is available.
set(PERFORMIX_GNU_TAR "")
find_program(_performix_tar NAMES gtar tar)
if(_performix_tar)
  execute_process(
    COMMAND "${_performix_tar}" --version
    OUTPUT_VARIABLE _performix_tar_version
    ERROR_QUIET
    OUTPUT_STRIP_TRAILING_WHITESPACE)
  if(_performix_tar_version MATCHES "GNU tar")
    set(PERFORMIX_GNU_TAR "${_performix_tar}")
  endif()
endif()

function(_performix_archive_command _out_commands _stage_dir _archive)
  cmake_parse_arguments(ARG "EXECUTABLE_ALL" "SOURCE_DIR;EXCLUDE" "" ${ARGN})

  set(_args
    "-DSTAGE_DIR=${_stage_dir}"
    "-DARCHIVE=${_archive}")
  if(ARG_SOURCE_DIR)
    list(APPEND _args "-DSOURCE_DIR=${ARG_SOURCE_DIR}")
  endif()
  if(ARG_EXCLUDE)
    list(APPEND _args "-DEXCLUDE=${ARG_EXCLUDE}")
  endif()
  if(ARG_EXECUTABLE_ALL)
    list(APPEND _args "-DEXECUTABLE_ALL=ON")
  endif()
  if(PERFORMIX_REPRODUCIBLE_ARCHIVES AND PERFORMIX_SOURCE_DATE_EPOCH)
    list(APPEND _args "-DSOURCE_DATE_EPOCH=${PERFORMIX_SOURCE_DATE_EPOCH}")
    if(PERFORMIX_GNU_TAR)
      list(APPEND _args "-DGNU_TAR=${PERFORMIX_GNU_TAR}")
    endif()
  endif()

  set(${_out_commands}
    COMMAND "${CMAKE_COMMAND}" ${_args} -P "${PERFORMIX_ARCHIVE_SCRIPT}"
    PARENT_SCOPE)
endfunction()

# ---------------------------------------------------------------------------
# performix_go_tool_bundle(...)
#
# Cross-compiles one Go tool for one target and packages it.
#
#   TOOL_NAME       Bundle name, for example apx-agent
#   MODULE_DIR      Directory to run go build from
#   PACKAGE         Package pattern, defaults to "."
#   GOOS/GOARCH     Go cross-compilation target
#   OS_LABEL        Bundle OS label, for example Linux
#   ARCH_LABEL      Bundle architecture label, for example aarch64
#   BINARY_NAME     Name inside the archive, defaults to TOOL_NAME
#   LDFLAGS         Contents of -ldflags, defaults to "-s -w"
#   EXTRA_FILE      Additional file to place in the archive
#   EXTRA_FILE_NAME Name for that file inside the archive
#   WATCH           Directories whose Go sources invalidate the bundle
#   DEPENDS         Extra file or target dependencies of the build step
#   ARCHIVE_VAR     Variable receiving the archive path
# ---------------------------------------------------------------------------

function(performix_go_tool_bundle)
  cmake_parse_arguments(ARG
    ""
    "TOOL_NAME;MODULE_DIR;PACKAGE;GOOS;GOARCH;OS_LABEL;ARCH_LABEL;BINARY_NAME;LDFLAGS;EXTRA_FILE;EXTRA_FILE_NAME;ARCHIVE_VAR"
    "WATCH;DEPENDS"
    ${ARGN})

  if(NOT ARG_BINARY_NAME)
    set(ARG_BINARY_NAME "${ARG_TOOL_NAME}")
  endif()
  if(NOT ARG_LDFLAGS)
    set(ARG_LDFLAGS "-s -w")
  endif()

  set(_slug "${ARG_TOOL_NAME}-${ARG_GOOS}-${ARG_GOARCH}")
  set(_binary_in_archive "${ARG_BINARY_NAME}")
  if(ARG_GOOS STREQUAL "windows")
    string(APPEND _binary_in_archive ".exe")
  endif()

  set(_build_dir "${CMAKE_CURRENT_BINARY_DIR}/build/${_slug}")
  set(_stage_dir "${CMAKE_CURRENT_BINARY_DIR}/stage/${_slug}")
  set(_binary "${_build_dir}/${_binary_in_archive}")
  set(_archive
    "${PERFORMIX_TOOLS_DIR}/${ARG_TOOL_NAME}/${PERFORMIX_ENGINE_VERSION}/${ARG_TOOL_NAME}-${ARG_OS_LABEL}-${ARG_ARCH_LABEL}.tar.gz")

  # Bundled tools are shipped to targets, so they are built without cgo and
  # stripped of debug information, matching the previous GoReleaser settings.
  performix_go_binary(
    OUTPUT "${_binary}"
    MODULE_DIR "${ARG_MODULE_DIR}"
    PACKAGE "${ARG_PACKAGE}"
    GOOS "${ARG_GOOS}"
    GOARCH "${ARG_GOARCH}"
    CGO OFF
    BUILD_FLAGS -trimpath
    LDFLAGS "${ARG_LDFLAGS}"
    WATCH ${ARG_WATCH}
    DEPENDS ${ARG_DEPENDS}
    COMMENT "Building ${ARG_TOOL_NAME} for ${ARG_GOOS}/${ARG_GOARCH}")

  set(_extra_commands)
  set(_extra_depends)
  if(ARG_EXTRA_FILE)
    if(NOT ARG_EXTRA_FILE_NAME)
      get_filename_component(ARG_EXTRA_FILE_NAME "${ARG_EXTRA_FILE}" NAME)
    endif()
    set(_extra_commands
      COMMAND "${CMAKE_COMMAND}" -E copy
              "${ARG_EXTRA_FILE}" "${_stage_dir}/${ARG_EXTRA_FILE_NAME}")
    set(_extra_depends "${ARG_EXTRA_FILE}")
  endif()

  _performix_archive_command(_archive_command "${_stage_dir}" "${_archive}" EXECUTABLE_ALL)

  add_custom_command(
    OUTPUT "${_archive}"
    COMMAND "${CMAKE_COMMAND}" -E rm -rf "${_stage_dir}"
    COMMAND "${CMAKE_COMMAND}" -E make_directory "${_stage_dir}"
    COMMAND "${CMAKE_COMMAND}" -E copy "${_binary}" "${_stage_dir}/${_binary_in_archive}"
    ${_extra_commands}
    ${_archive_command}
    DEPENDS "${_binary}" ${_extra_depends} "${PERFORMIX_ARCHIVE_SCRIPT}"
    COMMENT "Packaging ${ARG_TOOL_NAME}-${ARG_OS_LABEL}-${ARG_ARCH_LABEL}.tar.gz"
    VERBATIM)

  if(ARG_ARCHIVE_VAR)
    set(${ARG_ARCHIVE_VAR} "${_archive}" PARENT_SCOPE)
  endif()
endfunction()

# ---------------------------------------------------------------------------
# performix_payload_bundle(...)
#
# Packages a source directory verbatim, for tools that ship as sources rather
# than as a compiled binary.
#
#   TOOL_NAME    Bundle name
#   SOURCE_DIR   Directory whose contents become the archive root
#   OS_LABEL     Bundle OS label
#   ARCH_LABEL   Bundle architecture label
#   EXCLUDE      Pipe-separated list of path fragments to drop
#   ARCHIVE_VAR  Variable receiving the archive path
# ---------------------------------------------------------------------------

function(performix_payload_bundle)
  cmake_parse_arguments(ARG
    ""
    "TOOL_NAME;SOURCE_DIR;OS_LABEL;ARCH_LABEL;EXCLUDE;ARCHIVE_VAR"
    ""
    ${ARGN})

  set(_slug "${ARG_TOOL_NAME}-${ARG_OS_LABEL}-${ARG_ARCH_LABEL}")
  set(_stage_dir "${CMAKE_CURRENT_BINARY_DIR}/stage/${_slug}")
  set(_archive
    "${PERFORMIX_TOOLS_DIR}/${ARG_TOOL_NAME}/${PERFORMIX_ENGINE_VERSION}/${_slug}.tar.gz")

  file(GLOB_RECURSE _payload CONFIGURE_DEPENDS "${ARG_SOURCE_DIR}/*")

  _performix_archive_command(_archive_command "${_stage_dir}" "${_archive}"
    SOURCE_DIR "${ARG_SOURCE_DIR}"
    EXCLUDE "${ARG_EXCLUDE}")

  add_custom_command(
    OUTPUT "${_archive}"
    ${_archive_command}
    DEPENDS ${_payload} "${PERFORMIX_ARCHIVE_SCRIPT}"
    COMMENT "Packaging ${_slug}.tar.gz"
    VERBATIM)

  if(ARG_ARCHIVE_VAR)
    set(${ARG_ARCHIVE_VAR} "${_archive}" PARENT_SCOPE)
  endif()
endfunction()
