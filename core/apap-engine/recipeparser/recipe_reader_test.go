// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package recipeparser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe"
)

type mockRecipeReader struct {
	readRecipes func(func(string, error)) (map[string]recipe.Recipe, error)
	readRecipe  func(string) (recipe.Recipe, error)
	isValidFile func(string) bool
}

func (m mockRecipeReader) ReadRecipes(errHandler func(string, error)) (map[string]recipe.Recipe, error) {
	return m.readRecipes(errHandler)
}

func (m mockRecipeReader) ReadRecipe(file string) (recipe.Recipe, error) {
	return m.readRecipe(file)
}

func (m mockRecipeReader) IsRecipeValidFile(name string) bool {
	return m.isValidFile(name)
}

func TestParseRecipeHelperReturnsParseErrorForNamedRecipe(t *testing.T) {
	parseErr := errors.New("invalid recipe")
	reader := mockRecipeReader{
		readRecipes: func(errHandler func(string, error)) (map[string]recipe.Recipe, error) {
			errHandler("/tmp/code_hotspots.js", parseErr)
			return map[string]recipe.Recipe{}, nil
		},
		readRecipe: func(string) (recipe.Recipe, error) {
			return recipe.Recipe{}, nil
		},
		isValidFile: func(string) bool { return false },
	}

	_, err := ParseRecipeHelper(reader, "code_hotspots")
	require.Error(t, err)
	assert.ErrorIs(t, err, parseErr)
}

func TestParseRecipeHelperReturnsDoesNotExistWhenNoMatchingParseError(t *testing.T) {
	parseErr := errors.New("some other invalid recipe")
	reader := mockRecipeReader{
		readRecipes: func(errHandler func(string, error)) (map[string]recipe.Recipe, error) {
			errHandler("/tmp/other_recipe.js", parseErr)
			return map[string]recipe.Recipe{}, nil
		},
		readRecipe: func(string) (recipe.Recipe, error) {
			return recipe.Recipe{}, nil
		},
		isValidFile: func(string) bool { return false },
	}

	_, err := ParseRecipeHelper(reader, "code_hotspots")
	expectedErr := message.New(message.EngineRecipeDoesNotExist).WithMetadata(map[string]string{"recipe": "code_hotspots"})
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, message.ValidateMetadataPlaceholders(err))
}
