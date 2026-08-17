# Contributing

## Development

Run the full local check before submitting changes:

```sh
make check
```

For changes to parsers or shared state, also run:

```sh
go test -race ./...
```

## Parser fixtures

LM Studio CLI JSON can evolve. When adding support for a new schema, add a
minimal sanitized fixture/test that contains only the fields needed to exercise
the parser. Never commit real prompts, model responses, API tokens, private
paths, or other sensitive data.

## Metric compatibility

Prometheus metric and label names are part of the public interface. Avoid
renaming them after a release unless there is a clear migration plan.
