# lmstudio-exporter

A small Prometheus exporter for LM Studio / llmster. It runs alongside LM
Studio, reads machine-readable state from the `lms` CLI, and exposes metrics for
Prometheus without proxying inference traffic.

## What it collects

State is polled from:

- `lms daemon status --json` — daemon availability and PID.
- `lms ps --json` — loaded models, generation state, queue depth, and other
  model settings when LM Studio reports them.

Inference statistics are streamed from:

```text
lms log stream --source model --filter output --stats --json
```

The exporter extracts numerical statistics and intentionally does **not**
persist or expose prompts, responses, tool arguments, or request IDs.

## Platform status

| Platform | Status | Notes |
| --- | --- | --- |
| macOS Apple Silicon (arm64) | **Tested** | Validated with LM Studio/llmster and a real MLX inference workload. |
| macOS Intel (amd64) | Expected to work | Not currently tested. |
| Linux amd64 | Expected to work | Not currently tested. |
| Linux arm64 | Expected to work | Not currently tested. |

The exporter is written in portable Go and does not intentionally depend on
macOS-specific APIs. Linux support is therefore expected, provided the installed
LM Studio/llmster release exposes the same `lms` CLI commands and JSON fields.
However, Linux has **not yet been validated on real hardware**, so it should not
be considered a tested platform for this release.

Linux users who try the exporter are encouraged to report successful setups or
compatibility issues.

## Requirements

- LM Studio/llmster with the `lms` CLI installed.
- Go 1.23+ to build from source.
- LM Studio with JSON log streaming and prediction statistics support. LM
  Studio introduced log-stream JSON/stats in 0.3.26 and model generation/queue
  status in `lms ps --json` in 0.3.27.

## Build

```sh
git clone https://github.com/eXsoR65/lmstudio-exporter.git
cd lmstudio-exporter
make check
make build
```

The binary is written to `bin/lmstudio-exporter`.

## Run

Start LM Studio/llmster first, then run:

```sh
./bin/lmstudio-exporter
```

By default the exporter listens on `:9103` and exposes metrics at `/metrics`.

If `lms` is not available in the service's `PATH`, specify its absolute path:

```sh
./bin/lmstudio-exporter --lms.path="$HOME/.lmstudio/bin/lms"
```

Useful endpoints:

- `/metrics` — Prometheus metrics.
- `/healthz` — exporter process health.
- `/readyz` — returns HTTP 200 only after a successful poll reports llmster as
  running.

## Prometheus

Example scrape configuration:

```yaml
scrape_configs:
  - job_name: lmstudio
    scrape_interval: 15s
    static_configs:
      - targets:
          - mac-mini:9103
```

If Prometheus runs on another machine, ensure TCP/9103 is reachable only from
trusted monitoring hosts.

## Metrics

Core state metrics:

- `lmstudio_up`
- `lmstudio_loaded_models`
- `lmstudio_model_loaded{model,type}`
- `lmstudio_model_generating{model,type}`
- `lmstudio_model_queue_depth{model,type}`
- `lmstudio_model_context_length_tokens{model,type}`
- `lmstudio_model_parallelism{model,type}`
- `lmstudio_model_size_bytes{model,type}`

Inference metrics:

- `lmstudio_requests_total{model}`
- `lmstudio_input_tokens_total{model}`
- `lmstudio_output_tokens_total{model}`
- `lmstudio_reasoning_tokens_total{model}`
- `lmstudio_accepted_draft_tokens_total{model}`
- `lmstudio_input_tokens_last{model}`
- `lmstudio_output_tokens_last{model}`
- `lmstudio_context_tokens_last{model}`
- `lmstudio_context_utilization_ratio_last{model}`
- `lmstudio_tokens_per_second_last{model}`
- `lmstudio_tokens_per_second{model}` (histogram)
- `lmstudio_time_to_first_token_seconds{model}` (histogram)
- `lmstudio_model_load_seconds{model}` (histogram)
- `lmstudio_generation_seconds{model}` (histogram)

Exporter self-monitoring:

- `lmstudio_exporter_build_info`
- `lmstudio_exporter_poll_success`
- `lmstudio_exporter_last_poll_timestamp_seconds`
- `lmstudio_exporter_log_stream_up`
- `lmstudio_exporter_log_events_total`
- `lmstudio_exporter_log_parse_errors_total`

Some optional model and inference metrics appear only when the installed LM
Studio version includes the corresponding fields in its CLI JSON output.

The `*_last` token gauges describe the most recently completed prediction for
which LM Studio reported the relevant token counts. `lmstudio_context_tokens_last`
is input plus output tokens for that prediction.
`lmstudio_context_utilization_ratio_last` divides that value by the configured
model context length reported by `lms ps`.

This ratio is a **context/KV-cache pressure proxy**, not a measurement of actual
KV-cache memory allocation or live cache occupancy. It updates after a completed
prediction and is emitted only when the exporter has both input-token data and a
known context length for the model.

## Command-line options

```text
--web.listen-address=:9103
--web.telemetry-path=/metrics
--lms.path=lms
--lms.poll-interval=5s
--lms.command-timeout=4s
--lms.disable-log-stream=false
--log.level=info
--log.format=text
```

Use `--lms.disable-log-stream` if you only want daemon/model state metrics.

## macOS launchd

`packaging/launchd/io.github.exsor65.lmstudio-exporter.plist.example` is a
LaunchAgent template. Replace:

- `__EXPORTER_PATH__` with the absolute exporter binary path.
- `__LMS_PATH__` with the absolute `lms` path.

Then copy the rendered plist into `~/Library/LaunchAgents/` and load it with
`launchctl`. A LaunchAgent is intentional for the first release because `lms`
and llmster are normally installed in the user's environment.

## Design notes

The exporter intentionally does not sit in front of LM Studio's REST or
OpenAI-compatible endpoints. Existing clients continue talking directly to LM
Studio. `lmstudio-exporter` is observational only.

The CLI JSON schema is not fully documented. The parser accepts common
snake_case and camelCase field spellings and ignores unknown fields so routine
LM Studio additions do not break the exporter. `lmstudio_exporter_poll_success`
and `lmstudio_exporter_log_parse_errors_total` make schema incompatibilities
visible.

## License

MIT
