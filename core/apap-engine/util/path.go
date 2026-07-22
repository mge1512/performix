// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar"
)

var metachar = '*'
var unsupportedMetachars = "?[]{}"

// PathExists returns a bool indicating whether the specified path exists
// or not. If we fail to check existence of the file, we return the error.
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// IsChildPath reports whether childPath resolves to a path strictly below basePath.
// It returns false for basePath itself, parent paths, and paths that cannot be expressed as a relative child.
func IsChildPath(basePath string, childPath string) bool {
	rel, err := filepath.Rel(basePath, childPath)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}
	// If path escapes the base path
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	// Relative path must not be absolute
	if filepath.IsAbs(rel) {
		return false
	}
	return true
}

// RemapGlobbedPath composes a local concrete path by appending the path difference
// between the concrete remote path and the non-glob base of the remote glob.
// local - contains the local destination path
// remote - contains the expanded glob path
// remoteBase - contains the original remote path before expansion
//
// If remoteBase contains any glob, local MUST contain a single "*" at its end.
// That trailing "*" is removed before the delta is appended.
//
// Globs in remoteBase must be on the end of the path: "a/*/c" and "a/b/*c" are
// not valid.
//
// Only "*" is supported as a metacharacter. Paths cannot contain other
// metacharacters ("?[]{}"). Metacharacter escaping is not supported.
//
// Example:
//
//	local     = "tool/0/abc*"
//	remote    = "expanded/base/tool/0/abc-xyz/myFile"
//	remoteBase= "expanded/base/tool/0/abc*/**/*"
//
// Result:
//
//	"tool/0/abc-xyz/myFile"
func RemapGlobbedPath(local, remote, remoteBase string) (string, error) {
	// Clean and convert to slashes internally - will reconvert back to appropriate local path
	// separator before returning
	normalisedLocal := ForceToSlash(local)
	normalisedRemote := ForceToSlash(remote)
	normalisedRemoteBase := ForceToSlash(remoteBase)

	// Check that no disallowed metachars are present in any path, and that no paths are directories
	for _, p := range []string{normalisedLocal, normalisedRemote, normalisedRemoteBase} {
		if strings.ContainsAny(p, unsupportedMetachars) {
			return "", fmt.Errorf("'%v' contains unsupported meta character(s) ('%v')", p, unsupportedMetachars)
		}
		if strings.HasSuffix(p, "/") {
			return "", fmt.Errorf("'%v' is a directory", p)
		}
	}

	// Clean paths
	cleanLocal := ForceToSlash(filepath.Clean(normalisedLocal))
	cleanRemote := ForceToSlash(filepath.Clean(normalisedRemote))
	cleanRemoteBase := ForceToSlash(filepath.Clean(normalisedRemoteBase))

	// Check that remote is a concrete path (doesn't contain any metachars)
	if containsMetachar(cleanRemote) {
		return "", fmt.Errorf("remote path '%v' must be concrete (cannot contain '%v')", remote, string(metachar))
	}

	// Handle case where remoteBase is non-globbed
	if !containsMetachar(cleanRemoteBase) {
		if cleanRemote != cleanRemoteBase {
			return "", fmt.Errorf("remote '%v' and remote base '%v' are both concrete, but not the same", remote, remoteBase)
		}
		if containsMetachar(cleanLocal) {
			return "", fmt.Errorf("remote base is concrete, but local path '%v' is globbed", local)
		}
		return filepath.FromSlash(cleanLocal), nil
	}

	// Check that local ends in single metachar
	if strings.Count(cleanLocal, string(metachar)) != 1 || !strings.HasSuffix(cleanLocal, string(metachar)) {
		return "", fmt.Errorf("remote base '%v' is globbed, so local path '%v' must contain exactly 1 '%v', at its end", remoteBase, local, string(metachar))
	}

	// Check that metachars in remoteBase are strictly a suffix
	firstMetaCharIndex := strings.Index(cleanRemoteBase, string(metachar))
	if strings.ContainsFunc(cleanRemoteBase[firstMetaCharIndex+1:], func(r rune) bool {
		if r != metachar && r != '/' {
			return true
		}
		return false
	}) {
		return "", fmt.Errorf("remote base '%v' contains literal characters after the first '%v'", remoteBase, string(metachar))
	}

	// Validate remote under base
	match, err := doublestar.Match(cleanRemoteBase, cleanRemote)
	if err != nil {
		return "", err
	}
	if !match {
		return "", fmt.Errorf("remote path '%v' does not match remote base pattern '%v'", cleanRemote, cleanRemoteBase)
	}

	base := cleanRemoteBase[:firstMetaCharIndex]
	rel := strings.TrimPrefix(cleanRemote, base)

	deglobbedLocal := strings.TrimSuffix(cleanLocal, string(metachar))
	return filepath.FromSlash(filepath.Clean(deglobbedLocal + rel)), nil
}

func containsMetachar(p string) bool {
	return strings.Contains(p, string(metachar))
}

func ForceToSlash(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// MatchesAny checks whether the file path matches any of the specified globbed patterns.
// Note that escaping of metacharacters in the globbed patterns is not supported.
func MatchesAny(filePath string, patterns []string) (bool, error) {
	cleanedPath := ForceToSlash(filepath.Clean(ForceToSlash(filePath)))
	for _, pattern := range patterns {
		cleanedPattern := ForceToSlash(filepath.Clean(ForceToSlash(pattern)))
		shouldExclude, err := doublestar.Match(cleanedPattern, cleanedPath)
		if err != nil {
			return false, err
		}
		if shouldExclude {
			return true, nil
		}
	}
	return false, nil
}
