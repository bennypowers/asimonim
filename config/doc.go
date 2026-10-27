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
// design-tokens.yml, .config/design-tokens.json, or under the "asimonim",
// "designTokensLanguageServer", or "design-tokens-language-server" key in
// package.json. Those package.json keys are checked in that order. Load
// reads only the config file, LoadFull merges package.json with the config
// file, and LoadOrDefault and LoadFullOrDefault return Default when no
// config exists.
//
// LoadFull gives package.json precedence: when both sources set the same
// field, the package.json value wins, and the config file fills only fields
// that are unset in package.json. LoadFromPackageJSON reads only prefix,
// cdn, schema, groupMarkers, resolvers, and tokensFiles/files; other Config
// fields such as header and outputs are ignored when loading from
// package.json.
//
// LoadOrDefault and LoadFullOrDefault never return an error. They return
// Default when no configuration is found and also when loading fails,
// including malformed or unreadable configuration.
//
// Prefix is the global CSS variable prefix. GroupMarkers enables the draft
// convention where a group name can also be a token. Schema forces a schema
// version; when empty, detection proceeds from the document contents.
// Resolvers lists DTCG resolver documents and Outputs describes generated
// files.
package config
