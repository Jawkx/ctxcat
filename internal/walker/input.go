package walker

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// GetInputPaths returns a list of paths from command line arguments or stdin.
func GetInputPaths(args []string) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}

	// If no args, read from stdin
	var paths []string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				paths = append(paths, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading from stdin: %w", err)
		}
		return paths, nil
	}

	// No args and no pipe, default to current directory
	return []string{"."}, nil
}
