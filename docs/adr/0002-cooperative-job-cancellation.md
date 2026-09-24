# Use cooperative job cancellation

Kulee will expose cancellation as `POST /api/jobs/{id}/cancel`. Pending or running jobs transition atomically to a retained terminal `canceled` state; repeated cancellation is idempotent, while success, failed, and dead jobs return a conflict. Running handlers receive cancellation through the existing heartbeat-bounded context, so cancellation is cooperative and external side effects cannot be assumed reversible.
