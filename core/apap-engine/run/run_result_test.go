// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunResultIsFailure(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeSuccess, false},
		{RecipeInProgress, false},
		{RecipeInProgressPhase1Complete, false},
		{"failure_connect_to_daemon", true},
		{"failure_prepare_working_directory", true},
		{RecipeFailureCollect, true},
		{RecipeFailureProfiling, true},
		{RecipeFailureIdentify, true},
		{RecipeFailureRetrieve, true},
		{RecipeFailureRetrievePhase1Complete, true},
		{RecipeFailureIncompletePhase1Complete, true},
		{RecipeFailureDeploy, true},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultIsFailure(test.result))
		})
	}
}

func TestRunResultIsRunning(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeInProgress, true},
		{RecipeInProgressPhase1Complete, false},
		{RecipeSuccess, false},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultIsRunning(test.result))
		})
	}
}

func TestRunResultIsCompletingTransfer(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeInProgress, false},
		{RecipeInProgressPhase1Complete, true},
		{RecipeSuccess, false},
		{RecipeFailureRetrievePhase1Complete, false},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultIsCompletingTransfer(test.result))
		})
	}
}

func TestRunResultIsInProgress(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeInProgress, true},
		{RecipeInProgressPhase1Complete, true},
		{RecipeSuccess, false},
		{RecipeFailureRetrievePhase1Complete, false},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultIsInProgress(test.result))
		})
	}
}

func TestRunResultIsRenderable(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeSuccess, true},
		{RecipeInProgressPhase1Complete, true},
		{RecipeFailureRetrievePhase1Complete, true},
		{RecipeFailureIncompletePhase1Complete, true},
		{RecipeInProgress, false},
		{RecipeFailureRetrieve, false},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultIsRenderable(test.result))
		})
	}
}

func TestRunResultShouldShowFailure(t *testing.T) {
	tests := []struct {
		result   RunResult
		expected bool
	}{
		{RecipeSuccess, false},
		{RecipeInProgress, false},
		{RecipeInProgressPhase1Complete, false},
		{RecipeFailureRetrievePhase1Complete, false},
		{RecipeFailureIncompletePhase1Complete, false},
		{RecipeFailureRetrieve, true},
		{RecipeFailureProfiling, true},
	}

	for _, test := range tests {
		t.Run(string(test.result), func(t *testing.T) {
			assert.Equal(t, test.expected, RecipeResultShouldShowFailure(test.result))
		})
	}
}
