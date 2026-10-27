/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package config loads design tokens configuration.
//
// Configuration can live in .config/design-tokens.yaml, .config/
// design-tokens.yml, .config/design-tokens.json, or the designTokens field
// of package.json. Load reads only the config file, LoadFull merges
// package.json with the config file, and LoadOrDefault and LoadFullOrDefault
// return Default when no config exists.
//
// Prefix is the global CSS variable prefix. GroupMarkers enables the draft
// convention where a group name can also be a token. Schema forces a schema
// version; when empty, detection proceeds from the document contents.
// Resolvers lists DTCG resolver documents and Outputs describes generated
// files.
package config
