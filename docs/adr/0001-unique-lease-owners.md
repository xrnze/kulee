# Use unique lease owners per worker

Each worker receives a fresh opaque lease-owner token, and every lease renewal or terminal state update is fenced by that exact token. This prevents a stale worker from mutating a job after its lease has expired and the job has been reassigned, while preserving at-least-once processing.
