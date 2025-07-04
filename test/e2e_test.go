package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	binaryName = "ctxcat"
	binaryPath string
)

// TestMain is the entry point for testing. It compiles the binary once, runs all
// tests, and then cleans up.
func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", binaryName, "../main.go")
	buildOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to build binary: %v\n%s\n", err, string(buildOutput))
		os.Exit(1)
	}

	binaryPath, err = filepath.Abs(binaryName)
	if err != nil {
		fmt.Printf("Failed to get absolute path for binary: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()
	os.Remove(binaryPath)
	os.Exit(exitCode)
}

// run executes the compiled ctxcat binary.
func run(t *testing.T, args []string, stdin, workDir string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("Failed to run command: %v. Stderr: %s", err, stderr.String())
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// setupTestFS creates a temporary directory with a predefined file structure.
func setupTestFS(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	create := func(path, content string) {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)
	}

	create(".gitignore", "*.log\ndist/\n")
	create(".myignore", "*.md\n")
	create("file1.txt", "hello from file1")
	create("file2.md", "markdown content")
	create("ignored.log", "this is a log")
	create("binary_file", "a\x00b")
	create("docs/guide.md", "guide in docs")
	create("src/.gitignore", "component.js\n")
	create("src/main.go", "package main")
	create("src/component.js", "ignored js")
	create("dist/bundle.js", "minified bundle")

	return tempDir
}

// defaultTemplate generates the expected output using the application's real default template.
func defaultTemplate(path, content string) string {
	extWithDot := filepath.Ext(path)
	ext := strings.TrimPrefix(extWithDot, ".")
	p := filepath.ToSlash(path)
	return fmt.Sprintf(
		"=== File Start: %s ===\n```%s\n%s\n```\n=== File End: %s ===\n\n",
		p,
		ext,
		content,
		p,
	)
}

func TestE2E(t *testing.T) {
	testCases := []struct {
		name                string
		args                []string
		stdin               string
		workDirSetup        func(t *testing.T) string
		expectedStdout      func(workDir string) string
		expectedFile        string
		expectedFileContent func(workDir string) string
		expectedStderr      string
		expectedExitCode    int
	}{
		{
			name:         "gitignore is respected by default",
			args:         []string{"."},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate(".gitignore", "*.log\ndist/\n") +
					defaultTemplate(".myignore", "*.md\n") +
					defaultTemplate("docs/guide.md", "guide in docs") +
					defaultTemplate("file1.txt", "hello from file1") +
					defaultTemplate("file2.md", "markdown content") +
					defaultTemplate("src/.gitignore", "component.js\n") +
					defaultTemplate("src/main.go", "package main")
			},
			expectedExitCode: 0,
		},
		{
			name:         "no-gitignore flag",
			args:         []string{".", "--no-gitignore"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate(".gitignore", "*.log\ndist/\n") +
					defaultTemplate(".myignore", "*.md\n") +
					defaultTemplate("dist/bundle.js", "minified bundle") +
					defaultTemplate("docs/guide.md", "guide in docs") +
					defaultTemplate("file1.txt", "hello from file1") +
					defaultTemplate("file2.md", "markdown content") +
					defaultTemplate("ignored.log", "this is a log") +
					defaultTemplate("src/.gitignore", "component.js\n") +
					defaultTemplate("src/component.js", "ignored js") +
					defaultTemplate("src/main.go", "package main")
			},
			expectedExitCode: 0,
		},
		{
			name:         "exclude flag",
			args:         []string{".", "-e", "**/*.md", "-e", "src/**"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate(".gitignore", "*.log\ndist/\n") +
					defaultTemplate(".myignore", "*.md\n") +
					defaultTemplate("file1.txt", "hello from file1")
			},
			expectedExitCode: 0,
		},
		{
			name:         "custom ignore file",
			args:         []string{".", "--ignore-file", ".myignore"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate(".gitignore", "*.log\ndist/\n") +
					defaultTemplate(".myignore", "*.md\n") +
					defaultTemplate("file1.txt", "hello from file1") +
					defaultTemplate("src/.gitignore", "component.js\n") +
					defaultTemplate("src/main.go", "package main")
			},
			expectedExitCode: 0,
		},
		{
			name:         "custom template",
			args:         []string{"file1.txt", "--template", "Path: {path} | Content: {content}"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				// FIX: Removed the trailing `\n` because the app only adds it BETWEEN files.
				return "Path: file1.txt | Content: hello from file1"
			},
			expectedExitCode: 0,
		},
		{
			name:         "no recursive flag",
			args:         []string{".", "--no-recursive"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate(".gitignore", "*.log\ndist/\n") +
					defaultTemplate(".myignore", "*.md\n") +
					defaultTemplate("file1.txt", "hello from file1") +
					defaultTemplate("file2.md", "markdown content")
			},
			expectedExitCode: 0,
		},
		{
			name:         "include binary file",
			args:         []string{"binary_file", "--no-binary-check"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate("binary_file", "a\x00b")
			},
			expectedExitCode: 0,
		},
		{
			name:         "glob pattern",
			args:         []string{"**/*.md"},
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate("docs/guide.md", "guide in docs") +
					defaultTemplate("file2.md", "markdown content")
			},
			expectedExitCode: 0,
		},
		{
			name:         "stdin input",
			args:         []string{},
			stdin:        "file1.txt\ndocs/guide.md\nignored.log",
			workDirSetup: setupTestFS,
			expectedStdout: func(workDir string) string {
				return defaultTemplate("docs/guide.md", "guide in docs") +
					defaultTemplate("file1.txt", "hello from file1")
			},
			expectedExitCode: 0,
		},
		{
			name:           "output to file",
			args:           []string{"file1.txt", "-o", "output.txt"},
			workDirSetup:   setupTestFS,
			expectedStdout: func(workDir string) string { return "" },
			expectedFile:   "output.txt",
			expectedFileContent: func(workDir string) string {
				return defaultTemplate("file1.txt", "hello from file1")
			},
			expectedExitCode: 0,
		},
		{
			name:             "version flag",
			args:             []string{"--version"},
			workDirSetup:     setupTestFS,
			expectedStdout:   func(workDir string) string { return "1.0.0\n" },
			expectedExitCode: 0,
		},
		{
			name:             "invalid flag",
			args:             []string{"--nonexistent-flag"},
			workDirSetup:     setupTestFS,
			expectedStdout:   func(workDir string) string { return "" },
			expectedStderr:   "unknown flag: --nonexistent-flag",
			expectedExitCode: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			workDir := tc.workDirSetup(t)
			stdout, stderr, exitCode := run(t, tc.args, tc.stdin, workDir)

			if tc.expectedStdout != nil {
				expected := tc.expectedStdout(workDir)
				assert.Equal(t, expected, stdout, "stdout does not match expected")
			}

			if tc.expectedStderr != "" {
				assert.Contains(
					t,
					stderr,
					tc.expectedStderr,
					"stderr does not contain expected string",
				)
			}

			assert.Equal(t, tc.expectedExitCode, exitCode, "exit code does not match expected")

			if tc.expectedFile != "" {
				content, err := os.ReadFile(filepath.Join(workDir, tc.expectedFile))
				require.NoError(t, err, "could not read output file")
				expectedContent := tc.expectedFileContent(workDir)
				assert.Equal(
					t,
					expectedContent,
					string(content),
					"output file content does not match",
				)
			}
		})
	}
}
