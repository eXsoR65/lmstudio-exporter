package metrics

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	Store *Store
}

func (h Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	writeMetrics(w, h.Store.Snapshot())
}

func writeMetrics(w io.Writer, s Snapshot) {
	metricHeader(w, "lmstudio_exporter_build_info", "A metric with a constant '1' value labeled by exporter build information.", "gauge")
	fmt.Fprintf(w, "lmstudio_exporter_build_info{version=%s,commit=%s,build_date=%s} 1\n", label(s.Version), label(s.Commit), label(s.BuildDate))

	metricHeader(w, "lmstudio_up", "Whether the LM Studio llmster daemon reports itself as running.", "gauge")
	fmt.Fprintf(w, "lmstudio_up %d\n", boolFloat(s.DaemonUp))
	metricHeader(w, "lmstudio_daemon_pid", "PID reported by the LM Studio llmster daemon, or 0 when unavailable.", "gauge")
	fmt.Fprintf(w, "lmstudio_daemon_pid %d\n", s.DaemonPID)
	metricHeader(w, "lmstudio_exporter_poll_success", "Whether the most recent LM Studio CLI poll completed successfully.", "gauge")
	fmt.Fprintf(w, "lmstudio_exporter_poll_success %d\n", boolFloat(s.PollSuccess))
	metricHeader(w, "lmstudio_exporter_last_poll_timestamp_seconds", "Unix timestamp of the most recent LM Studio CLI poll attempt.", "gauge")
	fmt.Fprintf(w, "lmstudio_exporter_last_poll_timestamp_seconds %s\n", float(s.LastPollUnix))
	metricHeader(w, "lmstudio_exporter_log_stream_enabled", "Whether inference-stat collection from lms log stream is enabled.", "gauge")
	fmt.Fprintf(w, "lmstudio_exporter_log_stream_enabled %d\n", boolFloat(s.LogStreamEnabled))
	metricHeader(w, "lmstudio_exporter_log_stream_up", "Whether the lms log stream subprocess is currently connected.", "gauge")
	fmt.Fprintf(w, "lmstudio_exporter_log_stream_up %d\n", boolFloat(s.LogStreamUp))
	metricHeader(w, "lmstudio_exporter_log_events_total", "Number of JSON log stream events observed by the exporter.", "counter")
	fmt.Fprintf(w, "lmstudio_exporter_log_events_total %d\n", s.LogEventsTotal)
	metricHeader(w, "lmstudio_exporter_log_parse_errors_total", "Number of LM Studio log stream events that could not be parsed.", "counter")
	fmt.Fprintf(w, "lmstudio_exporter_log_parse_errors_total %d\n", s.LogParseErrors)

	metricHeader(w, "lmstudio_loaded_models", "Number of models currently reported as loaded by lms ps.", "gauge")
	fmt.Fprintf(w, "lmstudio_loaded_models %d\n", len(s.Models))

	modelMetricHeaders(w)
	for _, model := range s.Models {
		labels := fmt.Sprintf("model=%s,type=%s", label(model.Identifier), label(model.Type))
		fmt.Fprintf(w, "lmstudio_model_loaded{%s} 1\n", labels)
		fmt.Fprintf(w, "lmstudio_model_generating{%s} %d\n", labels, boolFloat(model.Generating))
		fmt.Fprintf(w, "lmstudio_model_queue_depth{%s} %s\n", labels, float(model.QueueDepth))
		if model.ContextLength > 0 {
			fmt.Fprintf(w, "lmstudio_model_context_length_tokens{%s} %s\n", labels, float(model.ContextLength))
		}
		if model.Parallel > 0 {
			fmt.Fprintf(w, "lmstudio_model_parallelism{%s} %s\n", labels, float(model.Parallel))
		}
		if model.SizeBytes > 0 {
			fmt.Fprintf(w, "lmstudio_model_size_bytes{%s} %s\n", labels, float(model.SizeBytes))
		}
	}

	inferenceMetricHeaders(w)
	models := sortedInferenceKeys(s.Inference)
	for _, model := range models {
		inf := s.Inference[model]
		labels := "model=" + label(model)
		fmt.Fprintf(w, "lmstudio_requests_total{%s} %d\n", labels, inf.RequestsTotal)
		fmt.Fprintf(w, "lmstudio_input_tokens_total{%s} %s\n", labels, float(inf.InputTokensTotal))
		fmt.Fprintf(w, "lmstudio_output_tokens_total{%s} %s\n", labels, float(inf.OutputTokensTotal))
		fmt.Fprintf(w, "lmstudio_reasoning_tokens_total{%s} %s\n", labels, float(inf.ReasoningTokensTotal))
		fmt.Fprintf(w, "lmstudio_accepted_draft_tokens_total{%s} %s\n", labels, float(inf.DraftTokensTotal))
		if inf.HasLastInputTokens {
			fmt.Fprintf(w, "lmstudio_input_tokens_last{%s} %s\n", labels, float(inf.LastInputTokens))
		}
		if inf.HasLastOutputTokens {
			fmt.Fprintf(w, "lmstudio_output_tokens_last{%s} %s\n", labels, float(inf.LastOutputTokens))
		}
		if inf.HasLastContextTokens {
			fmt.Fprintf(w, "lmstudio_context_tokens_last{%s} %s\n", labels, float(inf.LastContextTokens))
		}
		if inf.HasLastContextUtilization {
			fmt.Fprintf(w, "lmstudio_context_utilization_ratio_last{%s} %s\n", labels, float(inf.LastContextUtilizationRatio))
		}
		if inf.HasLastTPS {
			fmt.Fprintf(w, "lmstudio_tokens_per_second_last{%s} %s\n", labels, float(inf.LastTokensPerSecond))
		}
		writeHistogram(w, "lmstudio_tokens_per_second", labels, inf.TPS)
		writeHistogram(w, "lmstudio_time_to_first_token_seconds", labels, inf.TTFT)
		writeHistogram(w, "lmstudio_model_load_seconds", labels, inf.ModelLoad)
		writeHistogram(w, "lmstudio_generation_seconds", labels, inf.Generation)
	}
}

func modelMetricHeaders(w io.Writer) {
	metricHeader(w, "lmstudio_model_loaded", "Whether a model is currently loaded.", "gauge")
	metricHeader(w, "lmstudio_model_generating", "Whether a loaded model is currently generating.", "gauge")
	metricHeader(w, "lmstudio_model_queue_depth", "Number of prediction requests queued for a loaded model.", "gauge")
	metricHeader(w, "lmstudio_model_context_length_tokens", "Configured context length for a loaded model when reported by LM Studio.", "gauge")
	metricHeader(w, "lmstudio_model_parallelism", "Configured maximum concurrent predictions for a loaded model when reported by LM Studio.", "gauge")
	metricHeader(w, "lmstudio_model_size_bytes", "Loaded model size in bytes when reported by LM Studio.", "gauge")
}

func inferenceMetricHeaders(w io.Writer) {
	metricHeader(w, "lmstudio_requests_total", "Completed model prediction events observed from LM Studio.", "counter")
	metricHeader(w, "lmstudio_input_tokens_total", "Input tokens observed in completed predictions.", "counter")
	metricHeader(w, "lmstudio_output_tokens_total", "Output tokens observed in completed predictions.", "counter")
	metricHeader(w, "lmstudio_reasoning_tokens_total", "Reasoning output tokens observed in completed predictions when reported.", "counter")
	metricHeader(w, "lmstudio_accepted_draft_tokens_total", "Accepted speculative-decoding draft tokens observed when reported.", "counter")
	metricHeader(w, "lmstudio_input_tokens_last", "Input tokens in the most recently completed prediction that reported an input token count.", "gauge")
	metricHeader(w, "lmstudio_output_tokens_last", "Output tokens in the most recently completed prediction that reported an output token count.", "gauge")
	metricHeader(w, "lmstudio_context_tokens_last", "Input plus output tokens for the most recently completed prediction with input-token data and a known model context length.", "gauge")
	metricHeader(w, "lmstudio_context_utilization_ratio_last", "Context-pressure proxy for the most recently completed prediction: input plus output tokens divided by configured model context length.", "gauge")
	metricHeader(w, "lmstudio_tokens_per_second_last", "Token generation speed for the most recently completed prediction.", "gauge")
	metricHeader(w, "lmstudio_tokens_per_second", "Distribution of token generation speed for completed predictions.", "histogram")
	metricHeader(w, "lmstudio_time_to_first_token_seconds", "Distribution of time to first token for completed predictions.", "histogram")
	metricHeader(w, "lmstudio_model_load_seconds", "Distribution of per-request model load time when a load occurred and LM Studio reported it.", "histogram")
	metricHeader(w, "lmstudio_generation_seconds", "Distribution of total generation duration when reported by LM Studio.", "histogram")
}

func metricHeader(w io.Writer, name, help, typ string) {
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
	fmt.Fprintf(w, "# TYPE %s %s\n", name, typ)
}

func writeHistogram(w io.Writer, name, baseLabels string, h HistogramSnapshot) {
	if h.Count == 0 {
		return
	}
	for i, upper := range h.Buckets {
		fmt.Fprintf(w, "%s_bucket{%s,le=%s} %d\n", name, baseLabels, label(float(upper)), h.Counts[i])
	}
	fmt.Fprintf(w, "%s_bucket{%s,le=%s} %d\n", name, baseLabels, label("+Inf"), h.Count)
	fmt.Fprintf(w, "%s_sum{%s} %s\n", name, baseLabels, float(h.Sum))
	fmt.Fprintf(w, "%s_count{%s} %d\n", name, baseLabels, h.Count)
}

func boolFloat(v bool) int {
	if v {
		return 1
	}
	return 0
}

func float(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "+Inf"
	}
	if math.IsInf(v, -1) {
		return "-Inf"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func label(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\n", "\\n")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	return "\"" + v + "\""
}

func sortedInferenceKeys(m map[string]InferenceSnapshot) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
