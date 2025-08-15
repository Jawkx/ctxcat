![image](https://github.com/user-attachments/assets/18547d3b-49a4-421a-8e73-5d6cb97a53c9)

# ctxcat

A command-line utility that intelligently gathers and concatenates file contents into a single, formatted text blob - perfect for creating context-rich prompts for Large Language Models (LLMs).

## Features

- 🚀 **Simple & Fast**: Works out-of-the-box with sensible defaults
- 🎯 **Smart Filtering**: Automatically respects `.gitignore` and common ignore patterns
- 🔧 **Highly Customizable**: Powerful templating system for output formatting
- 🔗 **Unix-friendly**: Composable with other CLI tools via stdin/stdout
- 📁 **Flexible Input**: Supports files, directories, and glob patterns
- 🌐 **Cross-platform**: Consistent behavior across different operating systems

## Installation

### Quick and easy

```bash
npm install -g ctxcat
```

### Manual (recommended)

1. Download the latest release from the [releases page](https://github.com/Jawkx/ctxcat/releases).
2. Extract the binary to a directory in your `$PATH` (e.g. `/usr/local/bin`).

## Quick Start

```bash
# Concatenate all files in src/ directory
ctxcat src/

# Copy to clipboard (macOS)
ctxcat src/ | pbcopy

# Use glob patterns (recommended - quote them!)
ctxcat "src/**/*.js" "test/**/*.test.js"

# Combine with other tools
find . -name "*.md" | ctxcat
```

## Usage

```bash
ctxcat [OPTIONS] [PATH...]
```

### Arguments

- `PATH...`: Files, directories, or glob patterns to process
- If no paths provided, reads newline-separated file paths from stdin

### Options

#### Input & Path Control

| Flag | Description |
|------|-------------|
| `--no-recursive`, `-nr` | Disable recursive directory traversal |

#### Filtering & Ignoring

| Flag | Description |
|------|-------------|
| `--exclude <pattern>`, `-e <pattern>` | Exclude files matching glob pattern (repeatable) |
| `--no-gitignore` | Don't respect `.gitignore` files |
| `--ignore-file <filepath>` | Use custom ignore file (`.gitignore` syntax) |
| `--no-binary-check` | Don't skip binary files |

#### Output & Formatting

| Flag | Description |
|------|-------------|
| `--output <filepath>`, `-o <filepath>` | Write output to file instead of stdout |
| `--template <string>` | Custom output template (see templating section) |

#### Utility

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help message |
| `--version`, `-v` | Show version number |

## Output Templating

Customize the output format using the `--template` flag with these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{content}` | Full file content | `console.log('hello');` |
| `{path}` | Relative file path | `src/components/Button.js` |
| `{absPath}` | Absolute file path | `User/project/src/components/Button.js` |
| `{basename}` | Filename with extension | `Button.js` |
| `{filename}` | Filename without extension | `Button` |
| `{extension}` | File extension (no dot) | `js` |

### Default Template

`````
=== File Start: {path} ===
```{extension}
{content}
```
=== File End: {path} ===

`````

### Template Configuration

Create a `.ctxcat.template.txt` file for default templates. ctxcat searches for this file in:

1. Current working directory
2. User's home directory  
3. `~/.config/ctxcat/template.txt`

**Note:** The `--template` flag always overrides configuration files.

## Filtering Rules

Rules are applied in this order (first match wins):

1. **`--exclude` patterns** → Always excluded (highest priority)
2. **Custom ignore files** → Excluded if matched
3. **`.gitignore` files** → Excluded if matched (unless `--no-gitignore`)
4. **Default inclusion** → Included if no exclusion rules match

## Glob Patterns

ctxcat includes its own powerful glob engine with cross-platform support:

- `*` matches any characters except path separators
- `**` matches any characters including path separators (recursive)
- `?` matches any single character
- `[abc]` matches any character in brackets

### Important: Quote Your Globs!

**Recommended:**
```bash
ctxcat "src/**/*.js" "test/**/*.test.js"
```

**Also works (shell-expanded):**
```bash
ctxcat src/*.js test/*.js
```

Quoting ensures ctxcat's glob engine handles the pattern, providing consistent behavior across different shells and platforms.

## Examples

### Basic Usage

```bash
# Process all files in src directory
ctxcat src/

# Process specific files
ctxcat package.json README.md src/main.js

# Process with glob patterns
ctxcat "**/*.{js,ts,json}"
```

### Filtering & Exclusion

```bash
# Exclude specific patterns
ctxcat src/ -e "**/__pycache__/**" -e "*.log"

# Use custom ignore file
ctxcat . --ignore-file .contextignore

# Disable gitignore
ctxcat . --no-gitignore
```

### Custom Templates

```bash
# Simple template
ctxcat src/ --template "### {path}\n{content}\n---\n"

# Detailed template
ctxcat src/ --template "File: {path} ({extension})\n{content}\n\n"
```

### Piping & Composition

```bash
# Copy to clipboard
ctxcat src/ | pbcopy

# Save to file
ctxcat src/ -o context.txt

# Find specific files and process
find . -name "*.config.js" | ctxcat

# Chain with other tools
ctxcat src/ | wc -l
```

### Real-world Examples

```bash
# Create LLM context for a React project
ctxcat "src/**/*.{js,jsx,ts,tsx}" "*.{json,md}" --exclude "node_modules/**"

# Documentation files only
ctxcat "**/*.md" "docs/**/*.txt"

# Configuration files across project
find . -name "*.config.*" -o -name ".*rc*" | ctxcat
```

### Tips

1. Most UI doesn't support pasting large amount of text into their UI. You can create this bash script to create a temp file and then copy it's reference to your clipboard. Then you can paste it as file into your UI.

#### For macOS

``` bash
#!/bin/bash

# A script that saves stdout to a uniquely named file in a dedicated
# directory and copies it to the clipboard. It cleans up the *previously*
# created file on each new run.
#
# Usage:
#   some_command | ./this_script.sh

STORAGE_DIR="$HOME/.clipboard_files"
TRACKER_FILE="$STORAGE_DIR/last_file.txt"

mkdir -p "$STORAGE_DIR"

if [ -s "$TRACKER_FILE" ]; then
    LAST_FILE_PATH=$(cat "$TRACKER_FILE")
    if [ -f "$LAST_FILE_PATH" ]; then
        rm "$LAST_FILE_PATH"
    fi
fi

TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
NEW_FILE_PATH="$STORAGE_DIR/clip_${TIMESTAMP}.md"

cat - > "$NEW_FILE_PATH"

osascript -e "set the clipboard to (POSIX file \"$NEW_FILE_PATH\")"

echo "$NEW_FILE_PATH" > "$TRACKER_FILE"

echo ":white_check_mark: File copied to clipboard: ${NEW_FILE_PATH##*/}"
echo "   You can now paste it. The previous file has been cleaned up."
```

#### For Linux

``` bash
#!/bin/bash

# A script that saves stdout to a uniquely named file in a dedicated
# directory and copies it to the clipboard on Linux. It cleans up the
# previously created file on each new run.
#
# Usage:
#   some_command | ./this_script.sh

STORAGE_DIR="$HOME/.clipboard_files"
TRACKER_FILE="$STORAGE_DIR/last_file.txt"

check_dependencies() {
    if ! command -v xclip &> /dev/null
    then
        echo "Error: 'xclip' is not installed."
        echo "Please install it using your distribution's package manager, e.g.:"
        echo "  sudo apt install xclip   (for Debian/Ubuntu)"
        echo "  sudo yum install xclip   (for CentOS/RHEL)"
        echo "  sudo pacman -S xclip   (for Arch Linux)"
        exit 1
    fi
}

check_dependencies

mkdir -p "$STORAGE_DIR"

if [ -s "$TRACKER_FILE" ]; then
    LAST_FILE_PATH=$(cat "$TRACKER_FILE")
    if [ -f "$LAST_FILE_PATH" ]; then
        rm "$LAST_FILE_PATH"
    fi
fi

TIMESTAMP=$(date +"%Y-%m-%d_%H-%M-%S")
NEW_FILE_PATH="$STORAGE_DIR/clip_${TIMESTAMP}.md"

cat - > "$NEW_FILE_PATH"

xclip -selection clipboard < "$NEW_FILE_PATH"

echo "$NEW_FILE_PATH" > "$TRACKER_FILE"

echo ":white_check_mark: File copied to clipboard: ${NEW_FILE_PATH##*/}"
echo "    You can now paste it. The previous file has been cleaned up."
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT License](LICENSE)
