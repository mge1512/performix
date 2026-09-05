# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Go toolchain discovery, version gate and build helpers.
#
# This is the part that replaces mise.toml. mise pinned exact tool versions and
# downloaded them; here the pinned versions become minimum versions that are
# checked against whatever the host provides, so the toolchain stays under the
# control of the platform package manager.

include_guard(GLOBAL)

set(PERFORMIX_GO_MINIMUM_VERSION "1.26.6" CACHE STRING
    "Minimum accepted Go toolchain version")

set(PERFORMIX_GOTOOLCHAIN "local" CACHE STRING
    "Value of GOTOOLCHAIN for all Go invocations")

set(PERFORMIX_GOFLAGS "-mod=readonly" CACHE STRING
    "Value of GOFLAGS for all Go invocations")

set(PERFORMIX_GOPROXY "" CACHE STRING
    "Value of GOPROXY; leave empty to inherit the environment")

set(PERFORMIX_GOSUMDB "" CACHE STRING
    "Value of GOSUMDB; leave empty to inherit the environment")

set(PERFORMIX_GOMODCACHE "" CACHE PATH
    "Value of GOMODCACHE; leave empty to inherit the environment")

set(PERFORMIX_GOCACHE "" CACHE PATH
    "Value of GOCACHE; leave empty to inherit the environment")

# ---------------------------------------------------------------------------
# Locate the toolchain
# ---------------------------------------------------------------------------

find_program(PERFORMIX_GO_EXECUTABLE
  NAMES go
  HINTS ENV GOROOT
  PATH_SUFFIXES bin
  DOC "Path to the Go toolchain driver")

if(NOT PERFORMIX_GO_EXECUTABLE)
  message(FATAL_ERROR
    "No Go toolchain found.\n"
    "Install the Go package provided by your distribution, for example:\n"
    "  zypper install go1.26\n"
    "  apt install golang-1.26-go\n"
    "  dnf install golang\n"
    "Then re-run cmake, or point it at a specific toolchain with\n"
    "  -DPERFORMIX_GO_EXECUTABLE=/path/to/go")
endif()

function(_performix_go_env _out _name)
  execute_process(
    COMMAND "${PERFORMIX_GO_EXECUTABLE}" env "${_name}"
    OUTPUT_VARIABLE _value
    ERROR_VARIABLE _error
    RESULT_VARIABLE _rc
    OUTPUT_STRIP_TRAILING_WHITESPACE)
  if(NOT _rc EQUAL 0)
    message(FATAL_ERROR "'go env ${_name}' failed (${_rc}): ${_error}")
  endif()
  set(${_out} "${_value}" PARENT_SCOPE)
endfunction()

_performix_go_env(_performix_goversion GOVERSION)
_performix_go_env(PERFORMIX_GO_HOSTOS GOHOSTOS)
_performix_go_env(PERFORMIX_GO_HOSTARCH GOHOSTARCH)
_performix_go_env(PERFORMIX_GO_EXE GOEXE)

# GOVERSION is "go1.26.6" for a release and "devel go1.27-<hash> ..." otherwise.
if(NOT _performix_goversion MATCHES "go([0-9]+\\.[0-9]+(\\.[0-9]+)?)")
  message(FATAL_ERROR "Could not parse Go version from '${_performix_goversion}'")
endif()
set(PERFORMIX_GO_VERSION "${CMAKE_MATCH_1}")

if(PERFORMIX_GO_VERSION VERSION_LESS PERFORMIX_GO_MINIMUM_VERSION)
  message(FATAL_ERROR
    "Go ${PERFORMIX_GO_VERSION} at ${PERFORMIX_GO_EXECUTABLE} is too old.\n"
    "Performix requires Go ${PERFORMIX_GO_MINIMUM_VERSION} or later.\n"
    "Install a newer toolchain and point cmake at it with\n"
    "  -DPERFORMIX_GO_EXECUTABLE=/path/to/go\n"
    "or lower the gate with -DPERFORMIX_GO_MINIMUM_VERSION=<version> if you\n"
    "know the older toolchain is acceptable.")
endif()

# ---------------------------------------------------------------------------
# Shared environment
#
# GOTOOLCHAIN defaults to "local". With the upstream default of "auto", Go
# downloads and runs a toolchain matching the go directive in go.mod, which
# would silently defeat both the version gate above and the trust boundary of
# the installed toolchain.
# ---------------------------------------------------------------------------

set(PERFORMIX_GO_BASE_ENV
  "GOTOOLCHAIN=${PERFORMIX_GOTOOLCHAIN}"
  "GOFLAGS=${PERFORMIX_GOFLAGS}")

foreach(_var IN ITEMS GOPROXY GOSUMDB GOMODCACHE GOCACHE)
  if(PERFORMIX_${_var})
    list(APPEND PERFORMIX_GO_BASE_ENV "${_var}=${PERFORMIX_${_var}}")
  endif()
endforeach()

# ---------------------------------------------------------------------------
# performix_discover_go_modules(<out-var> <root>)
#
# Replaces core/scripts/go/all-modules.sh. Returns absolute directory paths of
# every Go module below <root>, ignoring vendor and testdata trees.
# ---------------------------------------------------------------------------

function(performix_discover_go_modules _out _root)
  file(GLOB_RECURSE _manifests CONFIGURE_DEPENDS "${_root}/go.mod")
  set(_dirs)
  foreach(_manifest IN LISTS _manifests)
    get_filename_component(_dir "${_manifest}" DIRECTORY)
    if(_dir MATCHES "/(vendor|testdata|node_modules)(/|$)")
      continue()
    endif()
    list(APPEND _dirs "${_dir}")
  endforeach()
  list(SORT _dirs)
  set(${_out} "${_dirs}" PARENT_SCOPE)
endfunction()

# ---------------------------------------------------------------------------
# performix_go_sources(<out-var> DIRS <dir>...)
#
# Collects the files that should invalidate a Go build. Go maintains its own
# build cache, so this list only has to be good enough to decide whether to
# invoke go build at all.
# ---------------------------------------------------------------------------

function(performix_go_sources _out)
  cmake_parse_arguments(ARG "" "" "DIRS" ${ARGN})
  set(_sources)
  foreach(_dir IN LISTS ARG_DIRS)
    file(GLOB_RECURSE _found CONFIGURE_DEPENDS
      "${_dir}/*.go"
      "${_dir}/go.mod"
      "${_dir}/go.sum")
    list(APPEND _sources ${_found})
  endforeach()
  list(REMOVE_DUPLICATES _sources)
  set(${_out} "${_sources}" PARENT_SCOPE)
endfunction()

# ---------------------------------------------------------------------------
# performix_go_binary(...)
#
# Declares a rule that produces one Go binary.
#
#   TARGET        Name of the custom target to create (optional)
#   OUTPUT        Absolute path of the binary to produce
#   MODULE_DIR    Directory to run go build from
#   PACKAGE       Package pattern to build, defaults to "."
#   GOOS/GOARCH   Cross-compilation target, defaults to the host
#   CGO           ON or OFF, defaults to OFF
#   TAGS          Build tags
#   LDFLAGS       Contents of a single -ldflags argument
#   BUILD_FLAGS   Extra go build flags, for example -trimpath
#   WATCH         Directories whose Go sources invalidate the output
#   DEPENDS       Extra file or target dependencies
#   COMMENT       Progress message
# ---------------------------------------------------------------------------

function(performix_go_binary)
  cmake_parse_arguments(ARG
    ""
    "TARGET;OUTPUT;MODULE_DIR;PACKAGE;GOOS;GOARCH;CGO;LDFLAGS;COMMENT"
    "TAGS;BUILD_FLAGS;WATCH;DEPENDS"
    ${ARGN})

  if(NOT ARG_OUTPUT OR NOT ARG_MODULE_DIR)
    message(FATAL_ERROR "performix_go_binary: OUTPUT and MODULE_DIR are required")
  endif()
  if(NOT ARG_PACKAGE)
    set(ARG_PACKAGE ".")
  endif()
  if(NOT ARG_CGO)
    set(ARG_CGO OFF)
  endif()

  set(_env ${PERFORMIX_GO_BASE_ENV})
  if(ARG_GOOS)
    list(APPEND _env "GOOS=${ARG_GOOS}")
  endif()
  if(ARG_GOARCH)
    list(APPEND _env "GOARCH=${ARG_GOARCH}")
  endif()
  if(ARG_CGO)
    list(APPEND _env "CGO_ENABLED=1")
    if(CMAKE_C_COMPILER)
      list(APPEND _env "CC=${CMAKE_C_COMPILER}")
    endif()
  else()
    list(APPEND _env "CGO_ENABLED=0")
  endif()

  set(_flags ${ARG_BUILD_FLAGS})
  if(ARG_TAGS)
    list(JOIN ARG_TAGS "," _tags)
    list(APPEND _flags "-tags=${_tags}")
  endif()
  if(ARG_LDFLAGS)
    list(APPEND _flags "-ldflags" "${ARG_LDFLAGS}")
  endif()

  performix_go_sources(_sources DIRS ${ARG_WATCH})

  get_filename_component(_outdir "${ARG_OUTPUT}" DIRECTORY)

  if(NOT ARG_COMMENT)
    get_filename_component(_name "${ARG_OUTPUT}" NAME)
    set(ARG_COMMENT "Building ${_name}")
  endif()

  add_custom_command(
    OUTPUT "${ARG_OUTPUT}"
    COMMAND "${CMAKE_COMMAND}" -E make_directory "${_outdir}"
    COMMAND "${CMAKE_COMMAND}" -E env ${_env}
            "${PERFORMIX_GO_EXECUTABLE}" build ${_flags}
            -o "${ARG_OUTPUT}" "${ARG_PACKAGE}"
    WORKING_DIRECTORY "${ARG_MODULE_DIR}"
    DEPENDS ${_sources} ${ARG_DEPENDS}
    COMMENT "${ARG_COMMENT}"
    VERBATIM)

  if(ARG_TARGET)
    add_custom_target(${ARG_TARGET} DEPENDS "${ARG_OUTPUT}")
  endif()
endfunction()

# ---------------------------------------------------------------------------
# performix_go_command(<target> COMMENT <text> ARGS <go-args>... [MODULES <dir>...])
#
# Runs one Go subcommand in every module. Replaces the for-each-go-module.sh
# family of scripts.
# ---------------------------------------------------------------------------

function(performix_go_command _target)
  cmake_parse_arguments(ARG "" "COMMENT" "ARGS;MODULES;DEPENDS" ${ARGN})

  if(NOT ARG_MODULES)
    set(ARG_MODULES ${PERFORMIX_GO_MODULES})
  endif()

  set(_commands)
  foreach(_module IN LISTS ARG_MODULES)
    list(APPEND _commands
      COMMAND "${CMAKE_COMMAND}" -E chdir "${_module}"
              "${CMAKE_COMMAND}" -E env ${PERFORMIX_GO_BASE_ENV}
              "${PERFORMIX_GO_EXECUTABLE}" ${ARG_ARGS})
  endforeach()

  add_custom_target(${_target}
    ${_commands}
    COMMENT "${ARG_COMMENT}"
    VERBATIM)

  if(ARG_DEPENDS)
    add_dependencies(${_target} ${ARG_DEPENDS})
  endif()
endfunction()
