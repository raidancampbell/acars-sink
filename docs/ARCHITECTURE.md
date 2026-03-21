# Architecture Outline

## Components

- **Listener**: TCP/UDP socket server; accepts NDJSON messages.
- **Decoder**: JSON -> Go struct (store raw + selected fields).
- **Storage**: SQLite writer with prepared inserts into raw and parsed tables.
- **Config**: flags/env for protocol, bind addr, db path.

## Suggested packages

```
/cmd/acars-sink      # main package
/internal/server     # listener (tcp/udp)
/internal/decoder    # JSON parsing
/internal/storage    # sqlite access
```

## Error handling strategy

- Log and continue on single-message parse errors.
- Retry on transient DB errors (basic backoff).
- Drop malformed messages after logging raw JSON.

## Future extensions

- Metrics endpoint (Prometheus).
- Message de-duplication.
- ACARS enrichment (aircraft registry, route lookup).
