package executil

import (
	"context"
	"testing"
	"time"
)

func TestRun_true(t *testing.T) {
	out, errOut, err := Run(context.Background(), 2*time.Second, "true")
	if err != nil {
		t.Fatalf("true: %v", err)
	}
	if errOut != "" {
		t.Fatalf("stderr: %q", errOut)
	}
	if out != "" {
		t.Fatalf("expected empty stdout, got %q", out)
	}
}
