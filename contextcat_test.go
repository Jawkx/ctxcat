package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain sets up the test environment before running tests
// and cleans up afterward.
func TestMain(m *testing.M) {
	// Build the binary to a temporary location for testing
	err := os.MkdirAll("tmp", 0755)
	if err != nil {
		panic("failed to create tmp dir for test binary")
	}
	cmd := exec.Command("go", "build", "-o", "tmp/contextgrep", ".")
	err = cmd.Run()
	if err != nil {
		panic("failed to build contextgrep for testing")
	}

	// Setup test data directory
	setupTestData()

	// Run the tests
	code := m.Run()

	// Cleanup
	cleanupTestData()
	os.RemoveAll("tmp")

	os.Exit(code)
}

func runCtxGrep(t *testing.T, args ...string) (string, string, error) {
	cmd := exec.Command("./tmp/contextgrep", args...)
	cmd.Dir = "testdata" // Run all tests from within the testdata directory
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func TestBasicRecursion(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "project1")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/main.go")
	assert.Contains(t, stdout, "File Start: project1/src/helper.go")
	assert.NotContains(t, stdout, "project1/dist/app")        // Ignored by .gitignore
	assert.NotContains(t, stdout, "project1/data.bin")        // Binary file
	assert.NotContains(t, stdout, "project1/secrets/key.txt") // Ignored by .gitignore
}

func TestNoRecursive(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "--no-recursive", "project1")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/main.go")
	assert.NotContains(t, stdout, "File Start: project1/src/helper.go") // Should be skipped
}

func TestGlobbing(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "project1/**/*.go")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/main.go")
	assert.Contains(t, stdout, "File Start: project1/src/helper.go")
	assert.NotContains(t, stdout, "README.md")
}

func TestExclude(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "-e", "**/*.md", "-e", "**/main.go", "project1")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/src/helper.go")
	assert.NotContains(t, stdout, "project1/README.md")
	assert.NotContains(t, stdout, "project1/main.go")
}

func TestNoGitignore(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "--no-gitignore", "project1")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/main.go")
	assert.Contains(t, stdout, "File Start: project1/dist/app") // Should now be included
}

func TestCustomIgnoreFile(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "--ignore-file", "custom.ignore", "project1")
	require.NoError(t, err)

	assert.NotContains(t, stdout, "File Start: project1/src/helper.go") // Excluded by custom.ignore
	assert.Contains(t, stdout, "File Start: project1/main.go")
}

func TestNoBinaryCheck(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "--no-binary-check", "project1/data.bin")
	require.NoError(t, err)

	assert.Contains(t, stdout, "File Start: project1/data.bin")
	assert.Contains(t, stdout, "binary content")
}

func TestCustomTemplate(t *testing.T) {
	template := "Path: {path} | Content:\n{content}\n---"
	stdout, _, err := runCtxGrep(t, "--template", template, "project1/README.md")
	require.NoError(t, err)

	assert.Equal(t, "Path: project1/README.md | Content:\n# Project 1\n---\n", stdout)
}

func TestOutputToFile(t *testing.T) {
	outputFilePath := filepath.Join("..", "tmp", "output.txt") // Place it outside testdata
	defer os.Remove(outputFilePath)

	_, _, err := runCtxGrep(t, "-o", outputFilePath, "project1/main.go")
	require.NoError(t, err)

	content, err := os.ReadFile(outputFilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "File Start: project1/main.go")
	assert.Contains(t, string(content), "package main")
}

func TestStdin(t *testing.T) {
	input := "project1/main.go\nproject1/README.md\n"
	cmd := exec.Command("./tmp/contextgrep")
	cmd.Dir = "testdata"
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, stderr.String())

	output := stdout.String()
	assert.Contains(t, output, "File Start: project1/main.go")
	assert.Contains(t, output, "File Start: project1/README.md")
	assert.NotContains(t, output, "helper.go")
}

func TestVersionFlag(t *testing.T) {
	stdout, _, err := runCtxGrep(t, "--version")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0\n", stdout)

	stdout, _, err = runCtxGrep(t, "-v")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0\n", stdout)
}
