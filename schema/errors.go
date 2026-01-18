/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package schema

import "errors"

// Sentinel errors for schema operations.
var (
	// ErrUnknownVersion indicates an unrecognized schema version.
	ErrUnknownVersion = errors.New("unknown schema version")

	// ErrMixedSchemas indicates tokens from different schema versions were mixed.
	ErrMixedSchemas = errors.New("mixed schema versions detected")

	// ErrInvalidToken indicates a token does not conform to the schema.
	ErrInvalidToken = errors.New("invalid token")

	// ErrMissingValue indicates a token is missing the required $value field.
	ErrMissingValue = errors.New("token missing $value")

	// ErrInvalidReference indicates a token reference is malformed.
	ErrInvalidReference = errors.New("invalid token reference")

	// ErrCircularReference indicates a circular reference was detected.
	ErrCircularReference = errors.New("circular reference detected")

	// ErrUnresolvedReference indicates a reference could not be resolved.
	ErrUnresolvedReference = errors.New("unresolved token reference")
)
