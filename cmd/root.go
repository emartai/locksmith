package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the root command and returns the process exit code.
func Execute(version, commit, date string) int {
	code, err := ExecuteArgs(os.Args[1:], os.Stdout, os.Stderr, version, commit, date)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return code
}

// ExecuteArgs runs the root command with explicit arguments and IO streams.
func ExecuteArgs(args []string, stdout, stderr io.Writer, version, commit, date string) (int, error) {
	exitCode := 0
	rootCmd := &cobra.Command{
		Use:           "locksmith",
		Short:         "Analyze Postgres migration files for dangerous operations",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("locksmith version %s (commit: %s, built: %s)", version, commit, date),
	}
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.AddCommand(newCheckCommand(stdout, stderr, &exitCode))
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		return 1, err
	}

	return exitCode, nil
}
