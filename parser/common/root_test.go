/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package common_test

import (
	"testing"

	"bennypowers.dev/asimonim/parser/common"
	"bennypowers.dev/asimonim/schema"
)

func TestIsRootToken(t *testing.T) {
	tests := []struct {
		name         string
		tokenName    string
		version      schema.Version
		groupMarkers []string
		want         bool
	}{
		{"v2025_10 $root", "$root", schema.V2025_10, nil, true},
		{"v2025_10 non-root", "primary", schema.V2025_10, nil, false},
		{"draft with matching marker", "$value", schema.Draft, []string{"$value"}, true},
		{"draft with non-matching marker", "primary", schema.Draft, []string{"$value"}, false},
		{"draft with empty markers", "primary", schema.Draft, nil, false},
		{"unknown version", "$root", schema.Unknown, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.IsRootToken(tt.tokenName, tt.version, tt.groupMarkers)
			if got != tt.want {
				t.Errorf("IsRootToken(%q, %v, %v) = %v, want %v",
					tt.tokenName, tt.version, tt.groupMarkers, got, tt.want)
			}
		})
	}
}

func TestGenerateRootTokenPath(t *testing.T) {
	groupPath := []string{"color", "brand"}
	result := common.GenerateRootTokenPath(groupPath, "$root", schema.V2025_10)

	if len(result) != len(groupPath) {
		t.Errorf("expected path length %d, got %d", len(groupPath), len(result))
	}
	for i, segment := range result {
		if segment != groupPath[i] {
			t.Errorf("path[%d] = %q, want %q", i, segment, groupPath[i])
		}
	}
}
