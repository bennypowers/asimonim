/*
Copyright 2026 Benny Powers. All rights reserved.
Use of this source code is governed by the GPLv3
license that can be found in the LICENSE file.
*/

package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bennypowers.dev/asimonim/cmd"
)

// testdataDir finds the testdata directory relative to this test file.
func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(filepath.Dir(wd), "testdata")
}

func captureAndExecute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	rootCmd := cmd.RootCmd
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = oldStdout

	return buf.String(), err
}

func TestValidateCommand(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	_, err := captureAndExecute(t, "validate", fixture)
	if err != nil {
		t.Errorf("validate command failed: %v", err)
	}
}

func TestValidateCommand_WithSchema(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/v2025_10/all-color-spaces/tokens.json")

	_, err := captureAndExecute(t, "validate", "--schema", "v2025.10", fixture)
	if err != nil {
		t.Errorf("validate command failed: %v", err)
	}
}

func TestValidateCommand_NonexistentFile(t *testing.T) {
	_, err := captureAndExecute(t, "validate", "/nonexistent/tokens.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestListCommand(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "list", fixture)
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
	if !strings.Contains(output, "color-primary") {
		t.Errorf("expected output to contain 'color-primary', got:\n%s", output)
	}
}

func TestListCommand_TypeFilter(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "list", "--type", "color", fixture)
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
	if strings.Contains(output, "dimension") {
		t.Errorf("expected no dimension tokens with color filter, got:\n%s", output)
	}
}

func TestListCommand_CSSFormat(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "list", "--format", "css", fixture)
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
	if !strings.Contains(output, ":root {") {
		t.Errorf("expected CSS output with :root selector, got:\n%s", output)
	}
}

func TestListCommand_MarkdownFormat(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "list", "--format", "markdown", fixture)
	if err != nil {
		t.Errorf("list command failed: %v", err)
	}
	if !strings.Contains(output, "##") {
		t.Errorf("expected markdown headings, got:\n%s", output)
	}
}

func TestSearchCommand(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "search", "primary", fixture)
	if err != nil {
		t.Errorf("search command failed: %v", err)
	}
	if !strings.Contains(output, "primary") {
		t.Errorf("expected output to contain 'primary', got:\n%s", output)
	}
}

func TestSearchCommand_Regex(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	output, err := captureAndExecute(t, "search", "--regex", "color-.*", fixture)
	if err != nil {
		t.Errorf("search command failed: %v", err)
	}
	if !strings.Contains(output, "color") {
		t.Errorf("expected output to contain 'color', got:\n%s", output)
	}
}

func TestSearchCommand_NameOnly(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	_, err := captureAndExecute(t, "search", "--name", "primary", fixture)
	if err != nil {
		t.Errorf("search command failed: %v", err)
	}
}

func TestVersionCommand(t *testing.T) {
	output, err := captureAndExecute(t, "version")
	if err != nil {
		t.Errorf("version command failed: %v", err)
	}
	if !strings.Contains(output, "asimonim") {
		t.Errorf("expected output to contain 'asimonim', got:\n%s", output)
	}
}

func TestVersionCommand_JSON(t *testing.T) {
	output, err := captureAndExecute(t, "version", "--format", "json")
	if err != nil {
		t.Errorf("version --format json failed: %v", err)
	}
	if !strings.Contains(output, "{") {
		t.Errorf("expected JSON output, got:\n%s", output)
	}
}

func TestConvertCommand_DTCG(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.json")

	_, err := captureAndExecute(t, "convert", "--format", "dtcg", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert command failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "$value") {
		t.Errorf("expected DTCG format with $value, got:\n%s", data)
	}
}

func TestConvertCommand_CSS(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.css")

	_, err := captureAndExecute(t, "convert", "--format", "css", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert command failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "--") {
		t.Errorf("expected CSS custom properties, got:\n%s", data)
	}
}

func TestConvertCommand_SCSS(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.scss")

	_, err := captureAndExecute(t, "convert", "--format", "scss", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert command failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "$") {
		t.Errorf("expected SCSS variables, got:\n%s", data)
	}
}

func TestConvertCommand_Stdout(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")

	// Explicitly pass --output="" to override any persisted state from previous tests
	output, err := captureAndExecute(t, "convert", "--format", "json", "--output", "", fixture)
	if err != nil {
		t.Errorf("convert to stdout failed: %v", err)
	}
	if !strings.Contains(output, "color") {
		t.Errorf("expected output to contain 'color', got:\n%s", output)
	}
}

func TestConvertCommand_Swift(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.swift")

	_, err := captureAndExecute(t, "convert", "--format", "swift", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert to swift failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "public enum") {
		t.Errorf("expected Swift enum, got:\n%s", data)
	}
}

func TestConvertCommand_Android(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.xml")

	_, err := captureAndExecute(t, "convert", "--format", "android", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert to android failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "<resources>") {
		t.Errorf("expected Android XML resources, got:\n%s", data)
	}
}

func TestConvertCommand_JS(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.ts")

	_, err := captureAndExecute(t, "convert", "--format", "js", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert to js failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "export") {
		t.Errorf("expected JS exports, got:\n%s", data)
	}
}

func TestConvertCommand_Snippets(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.code-snippets")

	_, err := captureAndExecute(t, "convert", "--format", "snippets", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert to snippets failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "prefix") {
		t.Errorf("expected snippet format with prefix, got:\n%s", data)
	}
}

func TestConvertCommand_Flatten(t *testing.T) {
	td := testdataDir(t)
	fixture := filepath.Join(td, "fixtures/draft/simple/tokens.json")
	outDir := t.TempDir()
	outFile := filepath.Join(outDir, "output.json")

	_, err := captureAndExecute(t, "convert", "--format", "dtcg", "--flatten", "--output", outFile, fixture)
	if err != nil {
		t.Fatalf("convert flatten failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if !strings.Contains(string(data), "color-primary") {
		t.Errorf("expected flattened keys, got:\n%s", data)
	}
}
