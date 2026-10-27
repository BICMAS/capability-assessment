# httpserver

Shared HTTP lifecycle helpers for BICMAS Learn Go services.

- Explicit server timeouts (`StandardTimeouts`, `LongRequestTimeouts`)
- Graceful `ListenAndServe` on SIGINT/SIGTERM
- Request wrap: request id, access log, body size limit, content-type checks

See `docs/http-lifecycle.md` for the service-wide contract.
