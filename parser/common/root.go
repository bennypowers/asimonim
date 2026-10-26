/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

package common

import (
	"slices"

	"bennypowers.dev/asimonim/schema"
)

// IsRootToken checks if a token name represents a root token for the given schema.
func IsRootToken(name string, version schema.Version, groupMarkers []string) bool {
	switch version {
	case schema.V2025_10:
		// In 2025.10, only "$root" is the reserved root token name
		return name == "$root"

	case schema.Draft:
		// In draft, use configured groupMarkers
		return slices.Contains(groupMarkers, name)

	default:
		return false
	}
}

// GenerateRootTokenPath generates the token path for a root token.
// Root tokens inherit the group path (don't add themselves to path).
func GenerateRootTokenPath(groupPath []string, rootTokenName string, version schema.Version) []string {
	return groupPath
}
