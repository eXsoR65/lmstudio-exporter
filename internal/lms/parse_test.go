package lms

import "testing"

func TestParseDaemonStatus(t *testing.T) {
	got, err := ParseDaemonStatus([]byte(`{"status":"running","pid":12345,"isDaemon":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Running || got.PID != 12345 {
		t.Fatalf("unexpected status: %+v", got)
	}
}

func TestParseModelsArray(t *testing.T) {
	input := `[
		{"identifier":"qwen/qwen3.5-4b","type":"LLM","isGenerating":true,"queuedPredictionCount":2,"contextLength":61952,"parallel":4,"size":"3.06 GB"},
		{"identifier":"text-embedding-model","type":"embedding","isGenerating":false,"queuedPredictionCount":0}
	]`
	models, err := ParseModels([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models", len(models))
	}
	m := models[0]
	if m.Identifier != "qwen/qwen3.5-4b" || !m.Generating || m.QueueDepth != 2 || m.ContextLength != 61952 || m.Parallel != 4 {
		t.Fatalf("unexpected model: %+v", m)
	}
	if m.SizeBytes < 3.05e9 || m.SizeBytes > 3.07e9 {
		t.Fatalf("unexpected size: %f", m.SizeBytes)
	}
}

func TestParseModelsWrappedSnakeCase(t *testing.T) {
	input := `{"loaded_models":[{"model_identifier":"test/model","model_type":"llm","is_generating":false,"queued_prediction_count":3,"context_length_tokens":8192}]}`
	models, err := ParseModels([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].Identifier != "test/model" || models[0].QueueDepth != 3 {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestParsePredictionStatsSnakeCase(t *testing.T) {
	input := `{
		"timestamp":"2026-08-17T12:00:00Z",
		"type":"llm.prediction.output",
		"modelIdentifier":"qwen/qwen3.5-4b",
		"output":"this content must never become a metric or log",
		"stats":{
			"input_tokens":4428,
			"total_output_tokens":298,
			"reasoning_output_tokens":11,
			"tokens_per_second":19.08,
			"time_to_first_token_seconds":0.79,
			"model_load_time_seconds":1.25
		}
	}`
	got, ok, err := ParsePredictionStats([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected stats event")
	}
	if got.Model != "qwen/qwen3.5-4b" || got.InputTokens != 4428 || got.OutputTokens != 298 || got.ReasoningTokens != 11 || got.TokensPerSecond != 19.08 || got.TimeToFirstToken != 0.79 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestParsePredictionStatsCamelCase(t *testing.T) {
	input := `{"type":"llm.prediction.output","modelIdentifier":"test/model","stats":{"tokensPerSecond":20.9,"timeToFirstToken":1.2,"totalTokensGenerated":65,"numberOfAcceptedDraftTokens":7,"totalTime":3.64}}`
	got, ok, err := ParsePredictionStats([]byte(input))
	if err != nil || !ok {
		t.Fatalf("err=%v ok=%v", err, ok)
	}
	if got.TokensPerSecond != 20.9 || got.OutputTokens != 65 || got.AcceptedDraftTokens != 7 || got.GenerationTime != 3.64 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestParsePredictionStatsLMStudioLogSchema(t *testing.T) {
	input := `{
		"timestamp":"2026-08-17T14:00:00-04:00",
		"type":"llm.prediction.output",
		"modelIdentifier":"qwen3.8-27b-mlx",
		"output":"private response content must be ignored",
		"stopReason":"eosFound",
		"tokensPerSecond":13.052745651997604,
		"timeToFirstTokenSec":1.324,
		"totalTimeSec":10.5,
		"promptTokensCount":4428,
		"predictedTokensCount":298,
		"totalTokensCount":4726
	}`
	got, ok, err := ParsePredictionStats([]byte(input))
	if err != nil || !ok {
		t.Fatalf("err=%v ok=%v", err, ok)
	}
	if got.Model != "qwen3.8-27b-mlx" || got.InputTokens != 4428 || got.OutputTokens != 298 || got.TokensPerSecond != 13.052745651997604 || got.TimeToFirstToken != 1.324 || got.GenerationTime != 10.5 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}

func TestParsePredictionStatsNoStats(t *testing.T) {
	_, ok, err := ParsePredictionStats([]byte(`{"type":"server.request","message":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("unexpected stats event")
	}
}

func TestParseModelsObjectKeyedByIdentifier(t *testing.T) {
	input := `{"models":{"qwen/keyed-model":{"type":"LLM","status":"generating","queuedPredictionRequests":1,"loadConfig":{"contextLength":32768,"maxConcurrentPredictions":2}}}}`
	models, err := ParseModels([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models", len(models))
	}
	m := models[0]
	if m.Identifier != "qwen/keyed-model" || !m.Generating || m.QueueDepth != 1 || m.ContextLength != 32768 || m.Parallel != 2 {
		t.Fatalf("unexpected model: %+v", m)
	}
}

func TestParsePredictionStatsNested(t *testing.T) {
	input := `{"event":{"type":"llm.prediction.output","model_instance_id":"nested/model","stats":{"input_tokens":12,"total_output_tokens":4,"tokens_per_second":18.5}}}`
	got, ok, err := ParsePredictionStats([]byte(input))
	if err != nil || !ok {
		t.Fatalf("err=%v ok=%v", err, ok)
	}
	if got.Model != "nested/model" || got.InputTokens != 12 || got.OutputTokens != 4 || got.TokensPerSecond != 18.5 {
		t.Fatalf("unexpected stats: %+v", got)
	}
}
