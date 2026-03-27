package main

import (
	"os"

	locksmithcmd "github.com/emartai/locksmith/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// pg_query_go requires CGO_ENABLED=1 for builds that use the parser.
	os.Exit(locksmithcmd.Execute(version, commit, date))
}
