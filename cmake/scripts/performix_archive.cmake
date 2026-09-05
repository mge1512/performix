# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

# Build-time helper that stages a directory and writes it out as a tar.gz.
#
# Invoked as:
#   cmake -DSTAGE_DIR=<dir> -DARCHIVE=<file.tar.gz>
#         [-DSOURCE_DIR=<dir>] [-DEXCLUDE=a|b|c] [-DEXECUTABLE_ALL=ON]
#         [-DSOURCE_DATE_EPOCH=<seconds>] [-DGNU_TAR=<path>]
#         -P performix_archive.cmake
#
# When SOURCE_DIR is given, STAGE_DIR is rebuilt from it, dropping any path that
# contains one of the EXCLUDE fragments. Otherwise STAGE_DIR is expected to have
# been populated by the caller.

cmake_minimum_required(VERSION 3.20)

if(NOT DEFINED STAGE_DIR OR NOT DEFINED ARCHIVE)
  message(FATAL_ERROR "performix_archive: STAGE_DIR and ARCHIVE are required")
endif()

# ---------------------------------------------------------------------------
# Stage
# ---------------------------------------------------------------------------

if(DEFINED SOURCE_DIR)
  if(NOT IS_DIRECTORY "${SOURCE_DIR}")
    message(FATAL_ERROR "performix_archive: source directory not found: ${SOURCE_DIR}")
  endif()

  file(REMOVE_RECURSE "${STAGE_DIR}")
  file(MAKE_DIRECTORY "${STAGE_DIR}")

  file(GLOB_RECURSE _entries RELATIVE "${SOURCE_DIR}" "${SOURCE_DIR}/*")
  foreach(_entry IN LISTS _entries)
    if(DEFINED EXCLUDE AND NOT EXCLUDE STREQUAL "" AND _entry MATCHES "${EXCLUDE}")
      continue()
    endif()
    get_filename_component(_dir "${_entry}" DIRECTORY)
    if(_dir)
      file(MAKE_DIRECTORY "${STAGE_DIR}/${_dir}")
    endif()
    file(COPY "${SOURCE_DIR}/${_entry}" DESTINATION "${STAGE_DIR}/${_dir}")
  endforeach()
endif()

if(NOT IS_DIRECTORY "${STAGE_DIR}")
  message(FATAL_ERROR "performix_archive: stage directory not found: ${STAGE_DIR}")
endif()

if(DEFINED EXECUTABLE_ALL AND EXECUTABLE_ALL)
  file(GLOB_RECURSE _staged "${STAGE_DIR}/*")
  if(_staged)
    file(CHMOD ${_staged} PERMISSIONS
      OWNER_READ OWNER_WRITE OWNER_EXECUTE
      GROUP_READ GROUP_EXECUTE
      WORLD_READ WORLD_EXECUTE)
  endif()
endif()

# ---------------------------------------------------------------------------
# Archive
# ---------------------------------------------------------------------------

file(GLOB _roots RELATIVE "${STAGE_DIR}" "${STAGE_DIR}/*")
if(NOT _roots)
  message(FATAL_ERROR "performix_archive: nothing staged in ${STAGE_DIR}")
endif()
list(SORT _roots)

get_filename_component(_archive_dir "${ARCHIVE}" DIRECTORY)
file(MAKE_DIRECTORY "${_archive_dir}")
file(REMOVE "${ARCHIVE}")

if(DEFINED SOURCE_DATE_EPOCH AND NOT SOURCE_DATE_EPOCH STREQUAL "")
  set(ENV{SOURCE_DATE_EPOCH} "${SOURCE_DATE_EPOCH}")
endif()

if(DEFINED GNU_TAR AND NOT GNU_TAR STREQUAL "")
  # GNU tar can pin ownership as well as timestamps. Compression runs through a
  # pipe, so the gzip header carries no name or timestamp of its own.
  execute_process(
    COMMAND "${GNU_TAR}"
            --format=gnu
            --sort=name
            --owner=0 --group=0 --numeric-owner
            "--mtime=@${SOURCE_DATE_EPOCH}"
            -czf "${ARCHIVE}"
            -- ${_roots}
    WORKING_DIRECTORY "${STAGE_DIR}"
    RESULT_VARIABLE _rc)
else()
  set(_tar_options --format=gnutar)
  if(DEFINED SOURCE_DATE_EPOCH AND NOT SOURCE_DATE_EPOCH STREQUAL "")
    string(TIMESTAMP _mtime "%Y-%m-%dT%H:%M:%SZ" UTC)
    list(APPEND _tar_options "--mtime=${_mtime}")
  endif()
  execute_process(
    COMMAND "${CMAKE_COMMAND}" -E tar czf "${ARCHIVE}" ${_tar_options} -- ${_roots}
    WORKING_DIRECTORY "${STAGE_DIR}"
    RESULT_VARIABLE _rc)
endif()

if(NOT _rc EQUAL 0)
  message(FATAL_ERROR "performix_archive: failed to write ${ARCHIVE} (exit ${_rc})")
endif()
