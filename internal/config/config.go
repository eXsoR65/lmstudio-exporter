package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"
)

type Config struct {
	ListenAddress    string
	TelemetryPath    string
	LMSPath          string
	PollInterval     time.Duration
	CommandTimeout   time.Duration
	DisableLogStream bool
	LogLevel         string
	LogFormat        string
	ShowVersion      bool
}

func Parse(args []string, output io.Writer) (Config, error) {
	cfg := Config{}
	fs := flag.NewFlagSet("lmstudio-exporter", flag.ContinueOnError)
	fs.SetOutput(output)

	fs.StringVar(&cfg.ListenAddress, "web.listen-address", ":9103", "Address to listen on for HTTP requests.")
	fs.StringVar(&cfg.TelemetryPath, "web.telemetry-path", "/metrics", "Path under which to expose Prometheus metrics.")
	fs.StringVar(&cfg.LMSPath, "lms.path", "lms", "Path to the LM Studio lms CLI.")
	fs.DurationVar(&cfg.PollInterval, "lms.poll-interval", 5*time.Second, "Interval for polling lms daemon status and lms ps.")
	fs.DurationVar(&cfg.CommandTimeout, "lms.command-timeout", 4*time.Second, "Timeout for short-lived lms commands.")
	fs.BoolVar(&cfg.DisableLogStream, "lms.disable-log-stream", false, "Disable inference metrics collected from lms log stream.")
	fs.StringVar(&cfg.LogLevel, "log.level", "info", "Log level: debug, info, warn, or error.")
	fs.StringVar(&cfg.LogFormat, "log.format", "text", "Log format: text or json.")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Print version information and exit.")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if cfg.PollInterval < time.Second {
		return Config{}, errors.New("lms.poll-interval must be at least 1s")
	}
	if cfg.CommandTimeout <= 0 {
		return Config{}, errors.New("lms.command-timeout must be greater than zero")
	}
	if cfg.TelemetryPath == "" || cfg.TelemetryPath[0] != '/' {
		return Config{}, errors.New("web.telemetry-path must begin with '/'")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("invalid log.level %q", cfg.LogLevel)
	}
	switch cfg.LogFormat {
	case "text", "json":
	default:
		return Config{}, fmt.Errorf("invalid log.format %q", cfg.LogFormat)
	}

	return cfg, nil
}
