package config

import (
	"bytes"
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	var out bytes.Buffer
	cfg, err := Parse(nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddress != ":9103" || cfg.PollInterval != 5*time.Second || cfg.LMSPath != "lms" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestRejectTooFastPoll(t *testing.T) {
	var out bytes.Buffer
	_, err := Parse([]string{"--lms.poll-interval=500ms"}, &out)
	if err == nil {
		t.Fatal("expected error")
	}
}
