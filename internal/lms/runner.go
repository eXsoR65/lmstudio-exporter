package lms

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type Runner struct {
	Path    string
	Timeout time.Duration
}

func (r Runner) run(ctx context.Context, args ...string) ([]byte, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, r.Path, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if cmdCtx.Err() != nil {
			return nil, fmt.Errorf("lms %v: %w", args, cmdCtx.Err())
		}
		msg := stderr.String()
		if len(msg) > 512 {
			msg = msg[:512]
		}
		return nil, fmt.Errorf("lms %v failed: %w: %s", args, err, msg)
	}
	return stdout.Bytes(), nil
}

func (r Runner) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	data, err := r.run(ctx, "daemon", "status", "--json")
	if err != nil {
		return DaemonStatus{}, err
	}
	return ParseDaemonStatus(data)
}

func (r Runner) Models(ctx context.Context) ([]Model, error) {
	data, err := r.run(ctx, "ps", "--json")
	if err != nil {
		return nil, err
	}
	return ParseModels(data)
}
