package executil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const DefaultTimeout = 12 * time.Second

// Run captures stdout and stderr from cmdName with args, killed after timeout.
func Run(ctx context.Context, timeout time.Duration, cmdName string, args ...string) (stdout, stderr string, err error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	return string(bytes.TrimSpace(outBuf.Bytes())), string(bytes.TrimSpace(errBuf.Bytes())), runErr
}

// LookPath wraps exec.LookPath with a clear error.
func LookPath(name string) (string, error) {
	p, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	return p, nil
}
