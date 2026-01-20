/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package validator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/validator"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read testdata/%s: %v", name, err)
	}
	return data
}

func TestValidateConsistency_ValidDraft(t *testing.T) {
	data := readTestdata(t, "valid-draft.json")
	errors := validator.ValidateConsistency(data, schema.Draft)

	if len(errors) != 0 {
		t.Errorf("expected no errors for valid draft, got %d: %v", len(errors), errors)
	}
}

func TestValidateConsistency_Valid2025(t *testing.T) {
	data := readTestdata(t, "valid-2025.json")
	errors := validator.ValidateConsistency(data, schema.V2025_10)

	if len(errors) != 0 {
		t.Errorf("expected no errors for valid 2025.10, got %d: %v", len(errors), errors)
	}
}

func TestValidateConsistency_DraftWithRef(t *testing.T) {
	data := readTestdata(t, "draft-with-ref.json")
	errors := validator.ValidateConsistency(data, schema.Draft)

	if len(errors) == 0 {
		t.Fatal("expected error for $ref in draft schema")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "$ref") {
			found = true
			if !strings.Contains(err.Suggestion, "curly-brace") && !strings.Contains(err.Suggestion, "2025.10") {
				t.Errorf("expected suggestion to mention curly-brace refs or 2025.10, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention $ref, got: %v", errors)
	}
}

func TestValidateConsistency_DraftWithExtends(t *testing.T) {
	data := readTestdata(t, "draft-with-extends.json")
	errors := validator.ValidateConsistency(data, schema.Draft)

	if len(errors) == 0 {
		t.Fatal("expected error for $extends in draft schema")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "$extends") {
			found = true
			if !strings.Contains(err.Suggestion, "2025.10") {
				t.Errorf("expected suggestion to mention 2025.10, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention $extends, got: %v", errors)
	}
}

func TestValidateConsistency_DraftWithStructuredColor(t *testing.T) {
	data := readTestdata(t, "draft-structured-color.json")
	errors := validator.ValidateConsistency(data, schema.Draft)

	if len(errors) == 0 {
		t.Fatal("expected error for structured color in draft schema")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "structured color") {
			found = true
			if !strings.Contains(err.Suggestion, "string") || !strings.Contains(err.Suggestion, "2025.10") {
				t.Errorf("expected suggestion to mention string format or 2025.10, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention structured color, got: %v", errors)
	}
}

func TestValidateConsistency_2025WithStringColor(t *testing.T) {
	data := readTestdata(t, "2025-string-color.json")
	errors := validator.ValidateConsistency(data, schema.V2025_10)

	if len(errors) == 0 {
		t.Fatal("expected error for string color in 2025.10 schema")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "string color") {
			found = true
			if !strings.Contains(err.Suggestion, "structured") {
				t.Errorf("expected suggestion to mention structured format, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention string color, got: %v", errors)
	}
}

func TestValidateConsistency_2025WithGroupMarkers(t *testing.T) {
	data := readTestdata(t, "2025-with-markers.json")
	errors := validator.ValidateConsistency(data, schema.V2025_10)

	if len(errors) == 0 {
		t.Fatal("expected error for group markers in 2025.10 schema")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "group marker") || strings.Contains(err.Message, "deprecated") {
			found = true
			if !strings.Contains(err.Suggestion, "$root") {
				t.Errorf("expected suggestion to mention $root, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention group marker, got: %v", errors)
	}
}

func TestValidateConsistency_ConflictingRoot(t *testing.T) {
	data := readTestdata(t, "conflicting-root.json")
	errors := validator.ValidateConsistency(data, schema.V2025_10)

	if len(errors) == 0 {
		t.Fatal("expected error for conflicting root patterns")
	}

	found := false
	for _, err := range errors {
		if strings.Contains(err.Message, "conflicting") {
			found = true
			if !strings.Contains(err.Suggestion, "$root") {
				t.Errorf("expected suggestion to mention $root, got: %s", err.Suggestion)
			}
		}
	}

	if !found {
		t.Errorf("expected error message to mention conflicting, got: %v", errors)
	}
}

func TestValidateConsistency_WithPath(t *testing.T) {
	data := readTestdata(t, "draft-with-ref.json")
	errors := validator.ValidateConsistencyWithPath(data, schema.Draft, "/path/to/tokens.json")

	if len(errors) == 0 {
		t.Fatal("expected error")
	}

	if errors[0].FilePath != "/path/to/tokens.json" {
		t.Errorf("expected FilePath to be /path/to/tokens.json, got: %s", errors[0].FilePath)
	}

	// Check that Error() includes the file path
	errStr := errors[0].Error()
	if !strings.Contains(errStr, "/path/to/tokens.json") {
		t.Errorf("expected Error() to include file path, got: %s", errStr)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      validator.ValidationError
		contains []string
	}{
		{
			name: "full error",
			err: validator.ValidationError{
				FilePath:   "tokens.json",
				Path:       "color.primary",
				Message:    "invalid value",
				Suggestion: "use correct format",
			},
			contains: []string{"tokens.json", "color.primary", "invalid value", "use correct format"},
		},
		{
			name: "no file path",
			err: validator.ValidationError{
				Path:       "color.primary",
				Message:    "invalid value",
				Suggestion: "use correct format",
			},
			contains: []string{"color.primary", "invalid value", "use correct format"},
		},
		{
			name: "no suggestion",
			err: validator.ValidationError{
				FilePath: "tokens.json",
				Path:     "color.primary",
				Message:  "invalid value",
			},
			contains: []string{"tokens.json", "color.primary", "invalid value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("expected Error() to contain %q, got: %s", s, errStr)
				}
			}
		})
	}
}
