package console

import (
	"context"
	"os"
)

// Run executes the console CLI logic. It returns an exit code appropriate for os.Exit.
func Run(args []string, stdout, stderr *os.File) int {
	cfg, err := parseFlags(args, stdout, stderr)
	if err != nil {
		return 2
	}
	return dispatchTrace(context.Background(), cfg, stdout, stderr)
}

// HTTP tracing implemented in pkg/http
