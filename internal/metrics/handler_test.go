package metrics

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eXsoR65/lmstudio-exporter/internal/lms"
)

func TestMetricsOutput(t *testing.T) {
	store := NewStore("0.1.0", "abc123", "2026-08-17T00:00:00Z")
	store.SetLogStreamEnabled(true)
	store.SetPoll(lms.DaemonStatus{Running: true, PID: 42}, []lms.Model{{Identifier: "qwen/test", Type: "LLM", Generating: true, QueueDepth: 2, ContextLength: 1000}}, true)
	store.ObservePrediction(lms.PredictionStats{
		Model:              "qwen/test",
		InputTokens:        100,
		HasInputTokens:     true,
		OutputTokens:       20,
		HasOutputTokens:    true,
		TokensPerSecond:    25,
		HasTokensPerSecond: true,
		TimeToFirstToken:   0.5,
		HasTTFT:            true,
	})

	var buf bytes.Buffer
	writeMetrics(&buf, store.Snapshot())
	out := buf.String()
	for _, want := range []string{
		"lmstudio_up 1",
		"lmstudio_exporter_log_stream_enabled 1",
		`lmstudio_model_generating{model="qwen/test",type="LLM"} 1`,
		`lmstudio_model_queue_depth{model="qwen/test",type="LLM"} 2`,
		`lmstudio_requests_total{model="qwen/test"} 1`,
		`lmstudio_input_tokens_total{model="qwen/test"} 100`,
		`lmstudio_input_tokens_last{model="qwen/test"} 100`,
		`lmstudio_output_tokens_last{model="qwen/test"} 20`,
		`lmstudio_context_tokens_last{model="qwen/test"} 120`,
		`lmstudio_context_utilization_ratio_last{model="qwen/test"} 0.12`,
		`lmstudio_tokens_per_second_last{model="qwen/test"} 25`,
		`lmstudio_time_to_first_token_seconds_count{model="qwen/test"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in metrics:\n%s", want, out)
		}
	}
}
