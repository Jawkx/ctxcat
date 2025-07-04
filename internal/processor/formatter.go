package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Formatter applies a template to a file's content and metadata.
type Formatter struct {
	template string
}

// NewFormatter creates a new formatter with a given template.
func NewFormatter(template string) (*Formatter, error) {
	return &Formatter{template: template}, nil
}

// Format reads a file and applies the loaded template.
func (f *Formatter) Format(path string) (string, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file %s: %w", path, err)
	}
	content := string(contentBytes)

	// Get relative path for output consistency
	relPath, err := filepath.Rel(".", path)
	if err != nil {
		relPath = path // Fallback to original path
	}
	relPath = filepath.ToSlash(relPath) // Use forward slashes for consistency

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path // Fallback to original path
	}

	absPath = filepath.ToSlash(absPath)

	base := filepath.Base(relPath)
	ext := filepath.Ext(base)
	filename := strings.TrimSuffix(base, ext)
	if len(ext) > 0 {
		ext = ext[1:] // Remove the leading dot
	}

	// Using strings.Replacer is efficient for multiple replacements.
	replacer := strings.NewReplacer(
		"{content}", content,
		"{path}", relPath,
		"{abspath}", absPath,
		"{basename}", base,
		"{filename}", filename,
		"{extension}", ext,
	)

	return replacer.Replace(f.template), nil
}
