# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.1.1] - 2026-08-17

### Fixed

- Parse the field names emitted by current LM Studio model-output logs: `promptTokensCount`, `predictedTokensCount`, `timeToFirstTokenSec`, and `totalTimeSec`.
- Recognize `acceptedDraftTokensCount` and additional token-count aliases for forward compatibility.

## [0.1.0] - 2026-08-17

### Added

- LM Studio/llmster availability from `lms daemon status --json`.
- Loaded-model state from `lms ps --json`.
- Completed-prediction statistics from `lms log stream --stats --json`.
- Prometheus metrics for requests, token counts, tokens/sec, TTFT, model load
  time, generation duration, model generation state, and queue depth.
- `/healthz` and `/readyz` endpoints.
- Privacy-preserving handling of model log events; raw prompt/response content
  is neither logged nor exported.
- launchd example for macOS.
- Unit tests, race-detector-safe state storage, CI workflow, and Makefile.
