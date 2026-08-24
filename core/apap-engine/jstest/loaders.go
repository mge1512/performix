// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package jstest

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/util"
)

var resolveJSFilePath = resolveJSFilePathImpl

func resolveJSFilePathImpl(t *testing.T, relativePath string) string {
	t.Helper()

	require.True(t, filepath.IsLocal(relativePath), "JS file path must be relative to apap-cli")

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to locate the jstest package")

	dir := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "apap-cli"))
	return util.CanonicalPath(filepath.Join(dir, relativePath))
}
