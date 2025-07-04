package processor

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/sabhiram/go-gitignore"
)

// Config holds the configuration for the file processor.
type Config struct {
	NoRecursive   bool
	NoGitignore   bool
	IgnoreFiles   []string
	ExcludeGlobs  []string
	NoBinaryCheck bool
}

// FileProcessor walks paths and filters files based on configuration.
type FileProcessor struct {
	config *Config

	// Cache for compiled .gitignore files to avoid re-parsing. Maps directory -> matcher.
	ignoreMatcherCache map[string]*ignore.GitIgnore

	// A single compiled matcher for all custom --ignore-file patterns.
	customIgnoreMatcher *ignore.GitIgnore

	mu sync.Mutex // Protects the cache
}

// New creates a new FileProcessor.
func New(config *Config) (*FileProcessor, error) {
	p := &FileProcessor{
		config:             config,
		ignoreMatcherCache: make(map[string]*ignore.GitIgnore),
		mu:                 sync.Mutex{},
	}

	// Pre-compile custom ignore files, as they apply to all paths globally.
	if len(config.IgnoreFiles) > 0 {
		var allPatterns []string
		for _, file := range config.IgnoreFiles {
			f, err := os.Open(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load ignore file %s: %v\n", file, err)
				continue
			}
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				allPatterns = append(allPatterns, scanner.Text())
			}
			f.Close()
		}
		if len(allPatterns) > 0 {
			p.customIgnoreMatcher = ignore.CompileIgnoreLines(allPatterns...)
		}
	}

	return p, nil
}

// ProcessPaths takes a list of initial paths/globs and returns a filtered list of files.
func (p *FileProcessor) ProcessPaths(paths []string) ([]string, error) {
	expandedPaths := make(map[string]struct{})
	for _, path := range paths {
		matches, err := doublestar.FilepathGlob(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid glob pattern '%s': %v\n", path, err)
			continue
		}
		for _, match := range matches {
			expandedPaths[match] = struct{}{}
		}
	}

	finalFiles := make(map[string]struct{})
	var mu sync.Mutex

	for path := range expandedPaths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			if p.shouldInclude(path) {
				mu.Lock()
				finalFiles[path] = struct{}{}
				mu.Unlock()
			}
			continue
		}

		isRecursivePattern := strings.Contains(path, "**")
		if p.config.NoRecursive && !isRecursivePattern {
			// If it's a directory and --no-recursive is on, we only check files in this dir, not subdirs.
			// This part of the logic is simplified; the main check in WalkDir handles recursion.
			// We let WalkDir start, but it won't go deep.
		}

		walkErr := filepath.WalkDir(path, func(currentPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				// To improve performance, skip directories that are ignored.
				if currentPath != path && p.shouldSkipDir(currentPath) {
					return filepath.SkipDir
				}
				return nil
			}

			if p.shouldInclude(currentPath) {
				mu.Lock()
				finalFiles[currentPath] = struct{}{}
				mu.Unlock()
			}
			return nil
		})

		if walkErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error walking %s: %v\n", path, walkErr)
		}
	}

	result := make([]string, 0, len(finalFiles))
	for file := range finalFiles {
		result = append(result, file)
	}
	return result, nil
}

// getGitignoreMatcher finds or compiles a .gitignore file for a given directory.
// It returns the matcher and a boolean indicating if one was found/created.
func (p *FileProcessor) getGitignoreMatcher(dir string) (*ignore.GitIgnore, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if matcher, ok := p.ignoreMatcherCache[dir]; ok {
		return matcher, matcher != nil
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	matcher, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		p.ignoreMatcherCache[dir] = nil
		return nil, false
	}

	p.ignoreMatcherCache[dir] = matcher
	return matcher, true
}

// isGitignored checks a path against the hierarchy of .gitignore files.
func (p *FileProcessor) isGitignored(path string) bool {
	if p.config.NoGitignore {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	currentDir := absPath
	info, err := os.Stat(absPath)
	if err == nil && !info.IsDir() {
		currentDir = filepath.Dir(absPath)
	}

	for {
		if matcher, ok := p.getGitignoreMatcher(currentDir); ok {
			relativePath, err := filepath.Rel(currentDir, absPath)
			if err != nil {
				continue
			}
			if matcher.MatchesPath(filepath.ToSlash(relativePath)) {
				return true
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return false
}

// shouldSkipDir checks if a directory should be skipped during the walk.
// This is a subset of shouldInclude, without the binary check.
func (p *FileProcessor) shouldSkipDir(path string) bool {
	// Precedence 1: --exclude flag
	if p.isExcluded(path) {
		return true
	}

	// Precedence 2: --ignore-file files
	if p.customIgnoreMatcher != nil {
		relPath, err := filepath.Rel(".", path)
		if err != nil {
			relPath = path
		}
		if p.customIgnoreMatcher.MatchesPath(filepath.ToSlash(relPath)) {
			return true
		}
	}

	// Precedence 3: .gitignore files
	if p.isGitignored(path) {
		return true
	}

	return false
}

// shouldInclude applies all filtering logic in the correct order of precedence for a file.
func (p *FileProcessor) shouldInclude(path string) bool {
	if p.shouldSkipDir(path) {
		return false
	}

	// Precedence 4: Binary file check
	if !p.config.NoBinaryCheck && isBinary(path) {
		return false
	}

	return true
}

// isExcluded checks if a path matches any of the --exclude glob patterns.
func (p *FileProcessor) isExcluded(path string) bool {
	for _, pattern := range p.config.ExcludeGlobs {
		match, err := doublestar.PathMatch(filepath.ToSlash(pattern), filepath.ToSlash(path))
		if err == nil && match {
			return true
		}
	}
	return false
}

// isBinary checks for null bytes in the first chunk of a file.
func isBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	return slices.Contains(buffer[:n], 0)
}
