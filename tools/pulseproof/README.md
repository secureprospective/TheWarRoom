# Pulse transport counter (development only)

Wrap a disposable fixture client's `http.RoundTripper` with `pulseproof.Counter` and
compare `Snapshot` before and after each step. Counts are completed HTTP responses
at this client's transport, grouped by workflow command; failed network attempts,
other clients and server-internal calls are not counted. `Unchanged` is counted only
when a response of at most 1 MiB is read to EOF and contains `unchanged: true`.
Unknown or non-replayable request bodies are grouped under `<unknown>`. The counter
is not server-wide instrumentation and must not be used for production accounting.
It stores no credentials or response bodies after consumption.

The fixture scenario runner is not implemented. A meaningful cache/ping/account-switch
scenario must exercise the frontend read owner and change subscription (L6 and L5);
conditional unchanged responses depend on L11. Do not infer cache or privacy guarantees
from this counter alone.
