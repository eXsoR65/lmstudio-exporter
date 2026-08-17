# Design

## Goals

`lmstudio-exporter` provides Prometheus telemetry for a local LM Studio/llmster
instance without becoming part of the inference request path.

## Data flow

```text
clients ───────────────────────────────> LM Studio / llmster
                                             │
                  ┌──────────────────────────┴──────────────────────────┐
                  │                                                     │
      lms daemon status --json                              lms log stream
      lms ps --json                              --source model --filter output
                  │                                      --stats --json
                  │                                                     │
                  └──────────────────────┬──────────────────────────────┘
                                         v
                                lmstudio-exporter
                                  :9103/metrics
                                         │
                                         v
                                    Prometheus
                                         │
                                         v
                                      Grafana
```

## Polling versus streaming

Daemon and model state are gauges, so they are polled every five seconds by
default. Prediction statistics are event data and are consumed continuously
from `lms log stream` so counters and histograms are updated once each completed
output event is emitted.

Prometheus scrapes an in-memory snapshot. A scrape never runs an `lms` command,
which keeps scrape latency independent of LM Studio CLI latency.

## Failure behavior

- If `lms daemon status` fails, `lmstudio_exporter_poll_success` becomes `0`.
- If llmster reports itself stopped, `lmstudio_up` becomes `0` and the current
  loaded-model set is cleared.
- If `lms ps` fails while llmster is running, the previous model snapshot is
  preserved and poll success becomes `0`.
- If the log-stream subprocess exits, its up metric becomes `0` and the exporter
  restarts it with bounded exponential backoff.
- JSON fields that are unknown are ignored.

## Privacy

`lms log stream` can include model output. Raw lines are kept only long enough
to decode one event. The exporter extracts a small allowlist of numerical
statistics and a model identifier. It never stores or exports prompt text,
response text, tool arguments, request IDs, or raw event bodies.

Stderr from the streaming subprocess is drained to `/dev/null` rather than
being copied into exporter logs.

## Labels

Only low-cardinality model identity/type labels are used. Request IDs, session
IDs, prompts, response IDs, stop strings, and tool names are deliberately not
Prometheus labels.

## Dependencies

Version 0.1 intentionally uses only the Go standard library. Prometheus' text
exposition format is generated directly. This keeps the initial binary small
and avoids introducing third-party runtime/build dependencies before the LM
Studio integration is proven against real headless workloads.
