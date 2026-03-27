package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/emartai/locksmith/internal/config"
	"github.com/emartai/locksmith/internal/parser"
	"github.com/emartai/locksmith/internal/reporter"
	"github.com/emartai/locksmith/internal/rules"
)

type checkOptions struct {
	databaseURL string
	configPath  string
	format      string
	severity    string
	noColor     bool
	output      string
}

func newCheckCommand(stdout, stderr io.Writer, exitCode *int) *cobra.Command {
	opts := &checkOptions{}

	cmd := &cobra.Command{
		Use:     "check [file|dir...]",
		Short:   "Analyze migration files for dangerous operations",
		Aliases: []string{"c"},
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := runCheck(stdout, stderr, args, opts)
			*exitCode = code
			return err
		},
	}

	cmd.Flags().StringVar(&opts.databaseURL, "database-url", "", "database connection placeholder for v1")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to locksmith.yml")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&opts.severity, "severity", "dangerous", "minimum severity to report: dangerous, warning, info")
	cmd.Flags().BoolVar(&opts.noColor, "no-color", false, "disable colored output")
	cmd.Flags().StringVar(&opts.output, "output", "", "write JSON output to a file instead of stdout")

	return cmd
}

func runCheck(stdout, stderr io.Writer, args []string, opts *checkOptions) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("check requires at least one file or directory")
	}

	if opts.databaseURL != "" {
		fmt.Fprintln(stderr, "database connection coming in v1")
	}

	cfg, err := config.LoadConfig(opts.configPath)
	if err != nil {
		return 1, err
	}

	files, warnings, progress, err := collectSQLFiles(args)
	if err != nil {
		return 1, err
	}
	for _, line := range progress {
		fmt.Fprintln(stderr, line)
	}
	for _, line := range warnings {
		fmt.Fprintln(stderr, line)
	}
	files = filterIgnoredPaths(files, cfg.IgnorePaths)

	if len(files) == 0 {
		return 0, nil
	}

	if opts.output != "" && opts.format != "json" {
		return 1, fmt.Errorf("--output is only supported with --format json")
	}

	minSeverity, err := parseMinimumSeverity(opts.severity)
	if err != nil {
		return 1, err
	}

	engine := rules.DefaultEngineWithOverrides(configRuleOverrides{rules: cfg.Rules})
	results := make([]reporter.JSONOutput, 0, len(files))
	allFindings := make(map[string][]rules.Finding, len(files))

	for _, file := range files {
		parseResult, err := parser.ParseFile(file)
		if err != nil {
			return 1, err
		}
		emitParseWarnings(stderr, file, *parseResult)

		findings := filterFindings(engine.Run(*parseResult), minSeverity)
		results = append(results, reporter.NewJSONOutput(file, findings))
		allFindings[file] = findings
	}

	if err := reporter.WriteGitHubSummary(results); err != nil {
		return 1, err
	}

	if opts.format == "json" {
		jsonWriter := stdout
		var outputFile *os.File
		if opts.output != "" {
			outputFile, err = os.Create(opts.output)
			if err != nil {
				return 1, fmt.Errorf("create output file %s: %w", opts.output, err)
			}
			defer outputFile.Close()
			jsonWriter = outputFile
		}

		if err := reporter.NewJSON(jsonWriter).Print(results); err != nil {
			return 1, fmt.Errorf("encode json output: %w", err)
		}
	} else {
		textReporter := reporter.New(stdout, !opts.noColor)
		for i, file := range files {
			if len(results) > 1 {
				if i > 0 {
					fmt.Fprintln(stdout)
				}
				fmt.Fprintf(stdout, "%s\n", file)
			}
			textReporter.Print(allFindings[file], file)
		}
		if len(results) > 1 {
			fmt.Fprintln(stdout)
			textReporter.PrintSummary(allFindings)
		}
	}

	return exitCodeForResults(results), nil
}

func collectSQLFiles(args []string) ([]string, []string, []string, error) {
	files := make([]string, 0)
	warnings := make([]string, 0)
	progress := make([]string, 0)
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("stat %s: %w", arg, err)
		}

		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(arg), ".sql") {
				files = append(files, filepath.Clean(arg))
			}
			continue
		}

		dirFiles := make([]string, 0)
		err = filepath.WalkDir(arg, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".sql") {
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				warnings = append(warnings, fmt.Sprintf("warning: skipping symlinked SQL file %s", filepath.Clean(path)))
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.Size() == 0 {
				warnings = append(warnings, fmt.Sprintf("warning: skipping empty SQL file %s", filepath.Clean(path)))
				return nil
			}
			dirFiles = append(dirFiles, filepath.Clean(path))
			return nil
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("walk %s: %w", arg, err)
		}
		if len(dirFiles) > 50 {
			progress = append(progress, fmt.Sprintf("Checking %s (%d files)...", filepath.Clean(arg), len(dirFiles)))
		}
		files = append(files, dirFiles...)
	}

	sort.Strings(files)
	files = slices.Compact(files)
	return files, warnings, progress, nil
}

func filterIgnoredPaths(files []string, ignorePaths []string) []string {
	if len(ignorePaths) == 0 {
		return files
	}

	filtered := make([]string, 0, len(files))
	for _, file := range files {
		if matchesIgnoredPath(file, ignorePaths) {
			continue
		}
		filtered = append(filtered, file)
	}
	return filtered
}

func emitParseWarnings(stderr io.Writer, file string, result rules.ParseResult) {
	for _, stmt := range result.Statements {
		if stmt.ParseError == "" {
			continue
		}
		fmt.Fprintf(stderr, "warning: parse error in %s: %s\n", file, stmt.ParseError)
	}
}

func matchesIgnoredPath(file string, ignorePaths []string) bool {
	normalizedFile := filepath.ToSlash(filepath.Clean(file))
	for _, ignoredPath := range ignorePaths {
		normalizedIgnore := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(ignoredPath)), "/")
		if normalizedIgnore == "." || normalizedIgnore == "" {
			continue
		}
		if strings.Contains(normalizedFile, normalizedIgnore) {
			return true
		}
	}
	return false
}

func parseMinimumSeverity(value string) (rules.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dangerous":
		return rules.SeverityDangerous, nil
	case "warning":
		return rules.SeverityWarning, nil
	case "info":
		return rules.SeverityInfo, nil
	default:
		return "", fmt.Errorf("invalid severity %q", value)
	}
}

func filterFindings(findings []rules.Finding, minSeverity rules.Severity) []rules.Finding {
	filtered := make([]rules.Finding, 0, len(findings))
	for _, finding := range findings {
		if shouldIncludeSeverity(finding.Severity, minSeverity) {
			filtered = append(filtered, finding)
		}
	}
	return filtered
}

func shouldIncludeSeverity(severity rules.Severity, minSeverity rules.Severity) bool {
	return severityRank(severity) <= severityRank(minSeverity)
}

func severityRank(severity rules.Severity) int {
	switch severity {
	case rules.SeverityDangerous:
		return 0
	case rules.SeverityWarning:
		return 1
	default:
		return 2
	}
}

func exitCodeForResults(results []reporter.JSONOutput) int {
	hasWarning := false
	for _, result := range results {
		for _, finding := range result.Findings {
			if finding.Severity == rules.SeverityDangerous {
				return 1
			}
			if finding.Severity == rules.SeverityWarning {
				hasWarning = true
			}
		}
	}

	if hasWarning {
		return 2
	}

	return 0
}

type configRuleOverrides struct {
	rules map[string]string
}

func (c configRuleOverrides) RuleOverrides() map[string]string {
	return c.rules
}
