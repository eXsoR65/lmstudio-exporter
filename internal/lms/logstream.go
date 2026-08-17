package lms

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"time"
)

type LogStreamCallbacks struct {
	OnStats      func(PredictionStats)
	OnState      func(bool)
	OnParseError func()
	OnEvent      func()
}

type LogStreamer struct {
	Path      string
	Logger    *slog.Logger
	Callbacks LogStreamCallbacks
}

func (s LogStreamer) Run(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := s.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if s.Callbacks.OnState != nil {
			s.Callbacks.OnState(false)
		}
		if err != nil && s.Logger != nil {
			s.Logger.Warn("lms log stream exited", "error", err, "restart_in", backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (s LogStreamer) runOnce(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.Path,
		"log", "stream",
		"--source", "model",
		"--filter", "output",
		"--stats",
		"--json",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start lms log stream: %w", err)
	}
	if s.Callbacks.OnState != nil {
		s.Callbacks.OnState(true)
	}

	// Drain stderr without logging its contents. This is intentional: LM Studio
	// logs may include prompt or response data, which the exporter must not persist.
	go func() { _, _ = io.Copy(io.Discard, stderr) }()

	scanner := bufio.NewScanner(stdout)
	// A completed model output can be present in the JSON line. Permit large
	// responses, but never store or log the raw line beyond parsing this event.
	scanner.Buffer(make([]byte, 64*1024), 32*1024*1024)
	for scanner.Scan() {
		if s.Callbacks.OnEvent != nil {
			s.Callbacks.OnEvent()
		}
		stats, ok, parseErr := ParsePredictionStats(scanner.Bytes())
		if parseErr != nil {
			if s.Callbacks.OnParseError != nil {
				s.Callbacks.OnParseError()
			}
			continue
		}
		if ok && s.Callbacks.OnStats != nil {
			s.Callbacks.OnStats(stats)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan lms log stream: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}
