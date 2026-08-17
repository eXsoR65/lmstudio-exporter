# Security

## Data handling

`lmstudio-exporter` is designed to export numerical telemetry only. LM Studio's
`lms log stream` can contain model output. The exporter parses each JSON event,
extracts recognized numeric statistics, and does not log or persist the raw
JSON line, prompt text, response text, tool arguments, or request identifiers.

The exporter deliberately drains `lms log stream` stderr without recording its
contents because LM Studio diagnostic output may also contain model data.

## Network exposure

The HTTP endpoint has no authentication or TLS. Bind it only to a trusted
interface/network, or place it behind an authenticated monitoring proxy. Model
identifiers are exposed as Prometheus labels.

## Reporting a vulnerability

Please report security issues privately to the repository maintainer rather
than opening a public issue containing sensitive details.
