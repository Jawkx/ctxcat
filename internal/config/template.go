package config

import (
	"os"
	"path/filepath"
)

const (
	// DefaultTemplate is used when no other template is found.
	// It is defined using string concatenation to allow for the use of backticks (`),
	// which are not permitted inside raw string literals.
	DefaultTemplate = "=== File Start: {path} ===\n" +
		"```{extension}\n" +
		"{content}\n" +
		"```\n" +
		"=== File End: {path} ===\n\n"

	templateFileName = ".contextgrep.template.txt"
)

// LoadTemplate finds and returns the template string to use based on precedence.
// Precedence: command-line flag > local file > home dir file > default.
func LoadTemplate(cliTemplate string) (string, error) {
	if cliTemplate != "" {
		return cliTemplate, nil
	}

	// Check current working directory
	cwd, err := os.Getwd()
	if err == nil {
		localTemplatePath := filepath.Join(cwd, templateFileName)
		if content, err := os.ReadFile(localTemplatePath); err == nil {
			return string(content), nil
		}
	}

	// Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homeTemplatePath := filepath.Join(homeDir, templateFileName)
		if content, err := os.ReadFile(homeTemplatePath); err == nil {
			return string(content), nil
		}

		// Check .config directory in home
		configTemplatePath := filepath.Join(homeDir, ".config", "contextgrep", "template.txt")
		if content, err := os.ReadFile(configTemplatePath); err == nil {
			return string(content), nil
		}
	}

	// Return the default if no custom template is found
	return DefaultTemplate, nil
}
