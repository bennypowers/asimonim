/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GNU General Public License
as published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
See the LICENSE file for details.
*/

// Package schema provides DTCG schema version handling.
//
// Version is an enum with three values: Unknown (the zero value), Draft,
// and V2025_10. String, URL, FromString, and FromURL convert between the
// enum and its external representations.
//
// # Zero value and auto-detection
//
// The zero value of Version is Unknown. Options fields such as
// parser.Options.SchemaVersion and load.Options.SchemaVersion use that zero
// value to mean "detect automatically". Do not pass Unknown to code that
// requires a concrete schema; let the parser or loader fill it in, or call
// DetectVersion yourself.
//
// DetectVersion checks, in order:
//
//  1. the $schema URL at the document root;
//  2. DetectionConfig.DefaultVersion, when supplied;
//  3. duck typing for unambiguous 2025.10 features such as $ref, $extends,
//     resolutionOrder, and structured color values;
//  4. Draft, for backward compatibility.
//
// An explicit non-Unknown Options.SchemaVersion always overrides detection.
//
// # Normalization
//
// NormalizeJSON converts JSONC with a UTF-8 BOM or trailing commas to JSON
// for callers that decode with encoding/json. YAML-aware normalization and
// comment masking are internal implementation details, not part of this
// package's API.
//
// # Style Dictionary
//
// Style Dictionary JSON is not DTCG. Detection and parsing operate on DTCG
// documents. Normalize Style Dictionary files into DTCG before handing them
// to this package.
package schema
