package cmd

import (
	"bufio"
	"fmt"
	"github.com/Jawkx/ctxcat/internal/config"
	"github.com/Jawkx/ctxcat/internal/processor"
	"github.com/Jawkx/ctxcat/internal/walker"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

var (
	noRecursive     bool
	excludePatterns []string
	noGitignore     bool
	ignoreFiles     []string
	noBinaryCheck   bool
	outputFile      string
	template        string
	showVersion     bool
)

const version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:   "contextgrep [OPTIONS] [PATH...]",
	Short: "Gathers file contents for LLM prompts.",
	Long: `contextgrep is a command-line utility that intelligently gathers and concatenates
file contents from specified paths into a single, formatted text blob.

Its primary purpose is to create a clean, context-rich string that can be easily
copied and pasted into a Large Language Model (LLM) prompt. It supports glob
patterns (including '**'), respects .gitignore files by default, and allows for
custom output formatting via templates.`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle version flag separately to avoid running the whole tool
		if showVersion {
			fmt.Println(version)
			return nil
		}

		// 1. Get input paths from arguments or stdin
		paths, err := walker.GetInputPaths(args)
		if err != nil {
			return fmt.Errorf("could not get input paths: %w", err)
		}

		// 2. Configure the file processor
		proc, err := processor.New(&processor.Config{
			NoRecursive:   noRecursive,
			NoGitignore:   noGitignore,
			IgnoreFiles:   ignoreFiles,
			ExcludeGlobs:  excludePatterns,
			NoBinaryCheck: noBinaryCheck,
		})
		if err != nil {
			return fmt.Errorf("failed to configure file processor: %w", err)
		}

		// 3. Process paths to get final list of files
		files, err := proc.ProcessPaths(paths)
		if err != nil {
			return fmt.Errorf("error processing paths: %w", err)
		}

		// 4. Sort files for deterministic output
		sort.Strings(files)

		// 5. Load the output template
		finalTemplate, err := config.LoadTemplate(template)
		if err != nil {
			return fmt.Errorf("could not load template: %w", err)
		}

		// 6. Set up the output writer
		var out io.Writer = os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("could not create output file %s: %w", outputFile, err)
			}
			defer f.Close()
			out = f
		}
		writer := bufio.NewWriter(out)
		defer writer.Flush()

		// 7. Format and write each file
		formatter, err := processor.NewFormatter(finalTemplate)
		if err != nil {
			return fmt.Errorf("failed to create formatter: %w", err)
		}

		for i, file := range files {
			formattedOutput, err := formatter.Format(file)
			if err != nil {
				// Log error to stderr and continue with other files
				fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", file, err)
				continue
			}
			if _, err := writer.WriteString(formattedOutput); err != nil {
				return fmt.Errorf("failed to write to output: %w", err)
			}
			// Add a newline between files if the template doesn't end with one
			// and it's not the last file.
			if len(formattedOutput) > 0 && formattedOutput[len(formattedOutput)-1] != '\n' &&
				i < len(files)-1 {
				writer.WriteRune('\n')
			}
		}

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Cobra already prints the error, so we just exit
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().
		BoolVarP(&noRecursive, "no-recursive", "r", false, "Disables recursive traversal of directories.")
	rootCmd.Flags().
		StringSliceVarP(&excludePatterns, "exclude", "e", nil, "A glob pattern for files or directories to exclude. Can be specified multiple times.")
	rootCmd.Flags().
		BoolVar(&noGitignore, "no-gitignore", false, "Do not respect the rules found in .gitignore files.")
	rootCmd.Flags().
		StringSliceVar(&ignoreFiles, "ignore-file", nil, "Path to a custom ignore file. Can be specified multiple times.")
	rootCmd.Flags().
		BoolVar(&noBinaryCheck, "no-binary-check", false, "Disable the binary file check.")
	rootCmd.Flags().
		StringVarP(&outputFile, "output", "o", "", "Write the output to a file instead of stdout.")
	rootCmd.Flags().
		StringVar(&template, "template", "", "A template string that defines the output format.")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show the version number.")

	// Set custom version template to just print the version string
	rootCmd.SetVersionTemplate(`{{.Version}}` + "\n")
}
