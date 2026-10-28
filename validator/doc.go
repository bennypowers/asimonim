/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package validator checks DTCG files for schema consistency.
//
// ValidateConsistency reports features that do not belong to the given
// schema version, such as $ref or $extends in a draft file, structured
// colors in a draft file, string colors in a 2025.10 file, and conflicting
// root token or group marker conventions. It returns a slice of
// ValidationError values; an empty slice means no problems were found.
//
// Pass the version you expect the file to use. To auto-detect a version
// first, use schema.DetectVersion.
//
// Style Dictionary JSON is not DTCG. Normalize it before validation.
package validator
