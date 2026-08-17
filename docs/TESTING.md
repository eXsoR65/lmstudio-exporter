# Testing on LM Studio

## 1. Confirm the required CLI sources

```sh
lms daemon status --json
lms ps --json
```

For inference statistics, start this in a separate terminal:

```sh
lms log stream --source model --filter output --stats --json
```

Send one normal inference request to LM Studio and confirm a newline-delimited
JSON output event appears. Stop the manual stream after the check.

## 2. Start the exporter

From a source build:

```sh
./bin/lmstudio-exporter --web.listen-address=127.0.0.1:9103
```

Or with the Apple Silicon release binary:

```sh
./lmstudio-exporter_0.1.0_darwin_arm64 \
  --lms.path="$HOME/.lmstudio/bin/lms" \
  --web.listen-address=127.0.0.1:9103
```

If `lms` is already in `PATH`, the `--lms.path` option can be omitted.

## 3. Verify state metrics

```sh
curl -fsS http://127.0.0.1:9103/readyz
curl -fsS http://127.0.0.1:9103/metrics | grep '^lmstudio_'
```

Expected baseline metrics include:

```text
lmstudio_up 1
lmstudio_exporter_poll_success 1
lmstudio_loaded_models 1
```

The exact loaded-model count depends on the local LM Studio state.

## 4. Verify inference metrics

Send a request through the client you normally use. No client configuration
change is required. Then inspect:

```sh
curl -fsS http://127.0.0.1:9103/metrics \
  | grep -E '^lmstudio_(requests|input_tokens|output_tokens|tokens_per_second|time_to_first_token)'
```

At minimum, after a completed inference and with a compatible LM Studio JSON
schema, `lmstudio_requests_total{model="..."}` should increase.

## 5. Check parser compatibility

These should remain zero during normal operation:

```text
lmstudio_exporter_poll_success 1
lmstudio_exporter_log_parse_errors_total 0
```

If `poll_success` is `0`, inspect the exporter log and the raw *schema* from
`lms ps --json`. If the log parse error counter increases, capture a sanitized
single JSON event from `lms log stream` with all prompt/response/tool content
removed. That fixture can be added to the parser tests without exposing model
content.

## 6. LAN Prometheus test

Once local testing passes, restart with:

```sh
./bin/lmstudio-exporter --web.listen-address=:9103
```

Then configure Prometheus using `examples/prometheus.yml`. The exporter does not
provide authentication or TLS, so TCP/9103 should only be reachable from the
trusted monitoring network.
