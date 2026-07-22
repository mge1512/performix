// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package util

import "testing"

func TestIsCatalogMetadataKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "metadata",
			key:  "metadata",
			want: true,
		},
		{
			name: "underscore-prefixed key",
			key:  "_schema",
			want: true,
		},
		{
			name: "message namespace",
			key:  "engine",
			want: false,
		},
		{
			name: "metadata-like namespace",
			key:  "metadataExtra",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCatalogMetadataKey(tt.key); got != tt.want {
				t.Fatalf("IsCatalogMetadataKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
