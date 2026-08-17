package lms

import (
	"fmt"
	"strings"
)

func ParseDaemonStatus(data []byte) (DaemonStatus, error) {
	v, err := decodeObject(data)
	if err != nil {
		return DaemonStatus{}, fmt.Errorf("decode daemon status: %w", err)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return DaemonStatus{}, fmt.Errorf("daemon status: expected object, got %s", debugShape(v))
	}

	status := strings.ToLower(firstString(m, "status"))
	running := status == "running"
	if isDaemonValue, ok := lookup(m, "isDaemon", "is_daemon"); ok {
		if isDaemon, ok := asBool(isDaemonValue); ok && status == "" {
			running = isDaemon
		}
	}
	pid := 0
	if f, ok := firstFloat(m, "pid"); ok {
		pid = int(f)
	}
	return DaemonStatus{Running: running, PID: pid}, nil
}

func ParseModels(data []byte) ([]Model, error) {
	v, err := decodeObject(data)
	if err != nil {
		return nil, fmt.Errorf("decode lms ps: %w", err)
	}
	objects := modelObjects(v)
	if len(objects) == 0 {
		// An empty array or a recognized empty models collection is valid.
		switch x := v.(type) {
		case []any:
			return []Model{}, nil
		case map[string]any:
			if child, ok := lookup(x, "models", "loadedModels", "loaded_models"); ok {
				if arr, ok := child.([]any); ok && len(arr) == 0 {
					return []Model{}, nil
				}
			}
		}
		return nil, fmt.Errorf("lms ps: no model objects found in %s", debugShape(v))
	}

	models := make([]Model, 0, len(objects))
	for _, m := range objects {
		identifier := firstString(m, "identifier", "modelIdentifier", "model_identifier", "modelKey", "model_key", "id")
		if identifier == "" {
			// Ignore metadata objects that were not actually model entries.
			continue
		}
		model := Model{
			Identifier:   identifier,
			Type:         firstString(m, "type", "modelType", "model_type"),
			Architecture: firstString(m, "architecture", "arch"),
		}
		if model.Type == "" {
			model.Type = "unknown"
		}
		if v, ok := lookupRecursive(m, "isGenerating", "is_generating", "generating", "generationStatus", "generation_status", "status"); ok {
			model.Generating, _ = asBool(v)
		}
		if f, ok := firstFloatRecursive(m, "queuedPredictionCount", "queued_prediction_count", "queuedPredictions", "queued_predictions", "queuedPredictionRequests", "queued_prediction_requests", "queueDepth", "queue_depth", "queuedRequests", "queued_requests", "queued"); ok {
			model.QueueDepth = f
		}
		if f, ok := firstFloatRecursive(m, "contextLength", "context_length", "contextLengthTokens", "context_length_tokens"); ok {
			model.ContextLength = f
		}
		if f, ok := firstFloatRecursive(m, "parallel", "parallelism", "maxConcurrentPredictions", "max_concurrent_predictions"); ok {
			model.Parallel = f
		}
		if size, ok := lookupRecursive(m, "sizeBytes", "size_bytes", "size"); ok {
			model.SizeBytes, _ = parseHumanBytes(size)
		}
		models = append(models, model)
	}
	if len(models) == 0 && len(objects) > 0 {
		return nil, fmt.Errorf("lms ps: model objects lacked a recognized identifier field")
	}
	return models, nil
}

func modelObjects(v any) []map[string]any {
	switch x := v.(type) {
	case []any:
		return mapsFromArray(x)
	case map[string]any:
		if child, ok := lookup(x, "models", "loadedModels", "loaded_models"); ok {
			switch collection := child.(type) {
			case []any:
				return mapsFromArray(collection)
			case map[string]any:
				return mapsFromObject(collection)
			}
		}
		// Some CLI schemas use object values keyed by identifier.
		var out []map[string]any
		for key, child := range x {
			m, ok := child.(map[string]any)
			if !ok {
				continue
			}
			if firstString(m, "identifier", "modelIdentifier", "model_identifier", "id") == "" && strings.Contains(key, "/") {
				m = cloneMap(m)
				m["identifier"] = key
			}
			out = append(out, m)
		}
		return out
	default:
		return nil
	}
}

func mapsFromObject(obj map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(obj))
	for key, item := range obj {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if firstString(m, "identifier", "modelIdentifier", "model_identifier", "id") == "" {
			m = cloneMap(m)
			m["identifier"] = key
		}
		out = append(out, m)
	}
	return out
}

func mapsFromArray(arr []any) []map[string]any {
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func cloneMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func ParsePredictionStats(data []byte) (PredictionStats, bool, error) {
	v, err := decodeObject(data)
	if err != nil {
		return PredictionStats{}, false, fmt.Errorf("decode log event: %w", err)
	}
	root, ok := v.(map[string]any)
	if !ok {
		return PredictionStats{}, false, nil
	}

	statsMap := root
	if rawStats, ok := lookupRecursive(root, "stats", "predictionStats", "prediction_stats", "metrics"); ok {
		if m, ok := rawStats.(map[string]any); ok {
			statsMap = m
		}
	}

	stats := PredictionStats{}
	if value, ok := lookup(root, "modelIdentifier", "model_identifier", "modelInstanceId", "model_instance_id", "model", "identifier"); ok {
		stats.Model, _ = asString(value)
	}
	if stats.Model == "" {
		if value, ok := lookupRecursive(v, "modelIdentifier", "model_identifier", "modelInstanceId", "model_instance_id"); ok {
			stats.Model, _ = asString(value)
		}
	}
	if stats.Model == "" {
		stats.Model = "unknown"
	}

	stats.InputTokens, stats.HasInputTokens = metricFloat(statsMap, root,
		"input_tokens", "inputTokens", "prompt_tokens", "promptTokens", "promptTokenCount", "promptTokensCount", "prompt_token_count", "prompt_tokens_count")
	stats.OutputTokens, stats.HasOutputTokens = metricFloat(statsMap, root,
		"total_output_tokens", "totalOutputTokens", "output_tokens", "outputTokens", "totalTokensGenerated", "total_tokens_generated", "generatedTokens", "generated_tokens", "predictedTokensCount", "predicted_tokens_count")
	stats.ReasoningTokens, stats.HasReasoningTokens = metricFloat(statsMap, root,
		"reasoning_output_tokens", "reasoningOutputTokens", "reasoning_tokens", "reasoningTokens", "reasoningTokensCount", "reasoning_tokens_count")
	stats.AcceptedDraftTokens, stats.HasDraftTokens = metricFloat(statsMap, root,
		"accepted_draft_tokens", "acceptedDraftTokens", "acceptedDraftTokensCount", "accepted_draft_tokens_count", "numberOfAcceptedDraftTokens", "numAcceptedDraftTokens")
	stats.TokensPerSecond, stats.HasTokensPerSecond = metricFloat(statsMap, root,
		"tokens_per_second", "tokensPerSecond", "tokensPerSec", "tokens_per_sec")
	stats.TimeToFirstToken, stats.HasTTFT = metricFloat(statsMap, root,
		"time_to_first_token_seconds", "timeToFirstTokenSeconds", "timeToFirstTokenSec", "time_to_first_token_sec", "timeToFirstToken", "time_to_first_token", "ttftSeconds", "ttft_seconds")
	stats.ModelLoadTime, stats.HasModelLoadTime = metricFloat(statsMap, root,
		"model_load_time_seconds", "modelLoadTimeSeconds", "modelLoadTime", "model_load_time")
	stats.GenerationTime, stats.HasGenerationTime = metricFloat(statsMap, root,
		"generation_time_seconds", "generationTimeSeconds", "total_time_seconds", "totalTimeSeconds", "totalTimeSec", "total_time_sec", "totalTime", "total_time")

	hasAny := stats.HasInputTokens || stats.HasOutputTokens || stats.HasReasoningTokens ||
		stats.HasDraftTokens || stats.HasTokensPerSecond || stats.HasTTFT ||
		stats.HasModelLoadTime || stats.HasGenerationTime
	return stats, hasAny, nil
}

func metricFloat(primary, fallback map[string]any, aliases ...string) (float64, bool) {
	if f, ok := firstFloatRecursive(primary, aliases...); ok {
		return f, true
	}
	return firstFloatRecursive(fallback, aliases...)
}
