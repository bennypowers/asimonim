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
	defer func() { os.Stdout = oldStdout }()

	// Create a fresh command tree for each test to avoid flag state pollution
	rootCmd := cmd.NewRootCmd()
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read captured output: %v", readErr)
	}
	if closeErr := r.Close(); closeErr != nil {
		t.Fatalf("failed to close read pipe: %v", closeErr)
	}

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

	// No need for --output="" workaround: each test gets a fresh command tree
	output, err := captureAndExecute(t, "convert", "--format", "json", fixture)
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

func TestNewRootCmd_HasAllSubcommands(t *testing.T) {
	rootCmd := cmd.NewRootCmd()
	expectedCmds := []string{"convert", "list", "search", "validate", "version"}
	for _, name := range expectedCmds {
		found := false
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("NewRootCmd() missing subcommand %q", name)
		}
	}
}

func TestNewRootCmd_FlagIsolation(t *testing.T) {
	// Verify that two independently created commands don't share flag state
	cmd1 := cmd.NewRootCmd()
	cmd2 := cmd.NewRootCmd()

	cmd1.SetArgs([]string{"version"})
	cmd2.SetArgs([]string{"version"})

	// Set a flag on cmd1
	if err := cmd1.PersistentFlags().Set("schema", "draft"); err != nil {
		t.Fatalf("failed to set schema flag: %v", err)
	}

	// cmd2 should still have the default value
	val, err := cmd2.PersistentFlags().GetString("schema")
	if err != nil {
		t.Fatalf("failed to get schema flag: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty schema on fresh command, got %q", val)
	}
}

func TestExecute(t *testing.T) {
	// Exercise the Execute() function that main.go calls
	old := cmd.RootCmd
	defer func() { cmd.RootCmd = old }()

	cmd.RootCmd = cmd.NewRootCmd()
	cmd.RootCmd.SetArgs([]string{"version"})

	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()
	defer r.Close()

	err := cmd.Execute()

	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("failed to close pipe: %v", closeErr)
	}
	var buf bytes.Buffer
	if _, readErr := buf.ReadFrom(r); readErr != nil {
		t.Fatalf("failed to read captured output: %v", readErr)
	}

	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "asimonim") {
		t.Errorf("expected 'asimonim' in output, got: %s", buf.String())
	}
}
