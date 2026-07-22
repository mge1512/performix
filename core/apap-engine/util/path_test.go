// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bmatcuk/doublestar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandGlob_GlobBase_NoEffectWithNoGlob(t *testing.T) {
	got, err := RemapGlobbedPath(
		filepath.FromSlash("tool/0/dest.txt"),
		filepath.FromSlash("expanded/base/tool/0/src.txt"),
		filepath.FromSlash("expanded/base/tool/0/src.txt"),
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("tool/0/dest.txt"), got)
}

func TestIsChildPath(t *testing.T) {
	base := filepath.FromSlash("/tmp/runs")

	testCases := []struct {
		name      string
		childPath string
		want      bool
	}{
		{
			name:      "direct child",
			childPath: filepath.Join(base, "abcdef123456"),
			want:      true,
		},
		{
			name:      "nested child",
			childPath: filepath.Join(base, "abcdef123456", "metadata.json"),
			want:      true,
		},
		{
			name:      "hidden file child",
			childPath: filepath.Join(base, ".myFile"),
			want:      true,
		},
		{
			name:      "filename begins with two dots",
			childPath: filepath.Join(base, "..abc"),
			want:      true,
		},
		{
			name:      "filename contains backslash after two dots",
			childPath: filepath.Join(base, `..\abc`),
			want:      runtime.GOOS != "windows",
		},
		{
			name:      "nested hidden file child",
			childPath: filepath.Join(base, "abcdef123456", ".metadata.json"),
			want:      true,
		},
		{
			name:      "base path itself",
			childPath: base,
			want:      false,
		},
		{
			name:      "parent path",
			childPath: filepath.Dir(base),
			want:      false,
		},
		{
			name:      "sibling with shared prefix",
			childPath: filepath.FromSlash("/tmp/runs-not-really/abcdef123456"),
			want:      false,
		},
		{
			name:      "escaped path",
			childPath: filepath.Clean(filepath.Join(base, "..", "other")),
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsChildPath(base, tc.childPath))
		})
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		patterns []string
		want     bool
		wantErr  error
	}{
		{
			name:     "returns false when there are no patterns",
			filePath: "/a/b/c.txt",
			patterns: []string{},
			want:     false,
		},
		{
			name:     "returns true when any pattern matches",
			filePath: "/a/b/c.txt",
			patterns: []string{"/x/**/*.txt", "/a/**/*.txt"},
			want:     true,
		},
		{
			name:     "returns true for an exact match",
			filePath: "/a/b/c.txt",
			patterns: []string{"/nope", "/a/b/c.txt"},
			want:     true,
		},
		{
			name:     "returns false when no pattern matches",
			filePath: "/a/b/c.txt",
			patterns: []string{"/a/*.txt", "/x/**/*.txt"},
			want:     false,
		},
		{
			name:     "cleans path and pattern before matching",
			filePath: "a/b/../c.txt",
			patterns: []string{"a//c.txt"},
			want:     true,
		},
		{
			name:     "normalises backslash separators before matching",
			filePath: `a\b\c.txt`,
			patterns: []string{"a/**/c.txt"},
			want:     true,
		},
		{
			name:     "backslash case 2",
			filePath: `C:\\a\b\c.txt`,
			patterns: []string{`C:\\a\**\*`},
			want:     true,
		},
		{
			name:     "returns doublestar error for invalid pattern",
			filePath: "/a/b/c.txt",
			patterns: []string{"{"},
			want:     false,
			wantErr:  doublestar.ErrBadPattern,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MatchesAny(tc.filePath, tc.patterns)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestExpandGlob_GlobBase_RequiresAndStripsSingleTrailingStar(t *testing.T) {
	got, err := RemapGlobbedPath(
		filepath.FromSlash("tool/0/*"),
		filepath.FromSlash("expanded/base/tool/0/inner/myFile"),
		filepath.FromSlash("expanded/base/tool/0/**/*"),
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("tool/0/inner/myFile"), got) // stripped "/*", then + delta
}

func TestExpandGlob_GlobBase_RemoteEqualsBase_NoDelta_StripsStarOnly(t *testing.T) {
	got, err := RemapGlobbedPath(
		filepath.FromSlash("foo/bar/*"),
		filepath.FromSlash("a/b/c"),
		filepath.FromSlash("a/b/c*"),
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("foo/bar"), got) // only strip trailing "/*"
}

func TestExpandGlob_GlobBase_Err_WhenSrcDoesNotEndWithSingleStar(t *testing.T) {
	cases := []string{
		"tool/*/end", // star not at the very end
		"tool/0/**",  // double star not allowed
		"tool/0/*/",  // trailing slash after star
		"tool/0/*x",  // star followed by another char
		"tool/0/",    // no star at all
		"tool/0",     // no star at all
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			_, err := RemapGlobbedPath(
				filepath.FromSlash(src),
				filepath.FromSlash("expanded/base/tool/0/inner/myFile"),
				filepath.FromSlash("expanded/base/tool/0/**/*"),
			)
			require.Error(t, err)
		})
	}
}

func TestExpandGlob_RemoteNotUnderBase_Err(t *testing.T) {
	_, err := RemapGlobbedPath(
		filepath.FromSlash("tool/0/*"),
		filepath.FromSlash("elsewhere/tool/0/inner/myFile"),
		filepath.FromSlash("expanded/base/tool/0/**/*"),
	)
	require.ErrorContains(t, err, "does not match remote base pattern")
}

func TestExpandGlob_MidpathGlobBaseExtraction_StripsStarAndAppendsDelta(t *testing.T) {
	got, err := RemapGlobbedPath(
		filepath.FromSlash("local/base*"),
		filepath.FromSlash("r/base/fixed/deeper/filea"),
		filepath.FromSlash("r/base*/**/*"),
	)
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("local/base/fixed/deeper/filea"), got)
}

func TestExpandGlob_AbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		got, err := RemapGlobbedPath(
			filepath.FromSlash("c:/local/*"),
			filepath.FromSlash("/r/base/fixed/deeper/file.txt"),
			filepath.FromSlash("/r/base/fixed/**/*"),
		)
		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("c:/local/deeper/file.txt"), got)
	} else {
		got, err := RemapGlobbedPath(
			filepath.FromSlash("/local/*"),
			filepath.FromSlash("/r/base/fixed/deeper/file.txt"),
			filepath.FromSlash("/r/base/fixed/**/*"),
		)
		require.NoError(t, err)
		assert.Equal(t, filepath.FromSlash("/local/deeper/file.txt"), got)
	}
}

func TestRemapGlobbedPath(t *testing.T) {
	testCases := []struct {
		name       string
		local      string
		remote     string
		remoteBase string
		want       string
		wantErr    string
	}{
		{
			name:       "fails if any arg contains unsupported metachars",
			local:      "",
			remote:     "one/two/three",
			remoteBase: "one/two/?",
			wantErr:    "contains unsupported meta character",
		},
		{
			name:       "fails if any arg is a directory",
			local:      "a/*",
			remote:     "one/two/three/",
			remoteBase: "one/two/*",
			wantErr:    "is a directory",
		},
		{
			name:       "fails if remote is not a concrete path",
			local:      "a/*",
			remote:     "one/two/three/*",
			remoteBase: "one/two/*",
			wantErr:    "must be concrete",
		},
		{
			name:       "fails if remote and remote base are both concrete, but not the same",
			local:      "a/*",
			remote:     "one/two/three/four",
			remoteBase: "one/two/three",
			wantErr:    "not the same",
		},
		{
			name:       "fails if remote base is concrete, but local is globbed",
			local:      "a/*",
			remote:     "one/two/three",
			remoteBase: "one/two/three",
			wantErr:    "remote base is concrete, but local path",
		},
		{
			name:       "fails if remote base is globbed, but local is concrete",
			local:      "a/b",
			remote:     "one/two/three/four",
			remoteBase: "one/two/three/*",
			wantErr:    "must contain exactly 1 '*'",
		},
		{
			name:       "fails if local has more than 1 star",
			local:      "a/b/**/*",
			remote:     "one/two/three/four",
			remoteBase: "one/two/three/*",
			wantErr:    "must contain exactly 1 '*'",
		},
		{
			name:       "fails if remote base is globbed, but local doesn't end in a star",
			local:      "a/b/**/c",
			remote:     "one/two/three/four",
			remoteBase: "one/two/three/*",
			wantErr:    "must contain exactly 1 '*', at its end",
		},
		{
			name:       "fails if metachars in remotePath aren't exclusively a suffix",
			local:      "a/b/*",
			remote:     "one/two/three/four",
			remoteBase: "one/*/three/*",
			wantErr:    "contains literal characters after the first '*'",
		},
		{
			name:       "fails if remote doesn't match remoteBase (1)",
			local:      "a/b/*",
			remote:     "x/y",
			remoteBase: "one/two/*",
			wantErr:    "does not match remote base pattern",
		},
		{
			name:       "fails if remote doesn't match remoteBase (2)",
			local:      "a/b/*",
			remote:     "one/two/three/four",
			remoteBase: "one/two/*",
			wantErr:    "does not match remote base pattern",
		},
		{
			name:       "success case A",
			local:      "one/two/t*",
			remote:     "one/two/three/four/five/six/seven",
			remoteBase: "one/two/th*/**/*",
			want:       "one/two/tree/four/five/six/seven",
		},
		{
			name:       "success case B",
			local:      "one/two/*",
			remote:     "one/two/threefourfive",
			remoteBase: "one/two/th*",
			want:       "one/two/reefourfive",
		},
		{
			name:       "cleans paths",
			local:      "one/two*",
			remote:     "one/two/three/fourfive",
			remoteBase: "one/////two//thr*///**/*",
			want:       "one/twoee/fourfive",
		},
		{
			name:       "handles pre-glob base matching the globbed path prefix",
			local:      "some/thing_*",
			remote:     "some/thing/thing_b",
			remoteBase: "some/thing/*",
			want:       "some/thing_thing_b",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RemapGlobbedPath(filepath.FromSlash(tc.local), filepath.FromSlash(tc.remote), filepath.FromSlash(tc.remoteBase))
			if tc.wantErr != "" {
				assert.Error(t, err, "expected error, got nil")
				assert.ErrorContains(t, err, tc.wantErr, fmt.Sprintf("'%v' does not contain expected error '%v'", err, tc.wantErr))
			} else {
				assert.NoError(t, err, "expected no error, got '%v'", err)
			}

			if tc.want != "" {
				assert.Equal(t, filepath.FromSlash(tc.want), got, fmt.Sprintf("want '%v', got '%v'", tc.want, got))
			}
		})
	}
}

func TestRemapGlobbedPath_CrossPlatform(t *testing.T) {
	t.Run("force POSIX-style target succeeds", func(t *testing.T) {
		var localPath string
		var want string
		if runtime.GOOS == "windows" {
			localPath = `C:\one\two*`
			want = `C:\one\twoee\four\five`
		} else {
			localPath = "/one/two*"
			want = "/one/twoee/four/five"
		}
		got, err := RemapGlobbedPath(
			localPath,
			"one/two/three/four/five",
			"one/two/thr*/*/*",
		)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
	t.Run("force windows-style target succeeds", func(t *testing.T) {
		var localPath string
		var want string
		if runtime.GOOS == "windows" {
			localPath = `C:\some\path*`
			want = `C:\some\paththis\is\a\path.txt`
		} else {
			localPath = "/some/path*"
			want = "/some/paththis/is/a/path.txt"
		}
		got, err := RemapGlobbedPath(
			localPath,
			`C:\this\is\a\path.txt`,
			`C:\**\*`,
		)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
