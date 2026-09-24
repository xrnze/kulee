# Kulee

Kulee coordinates durable jobs whose execution may be retried after worker failure or lease expiry.

## Language

**Lease owner**:
The worker instance currently authorized to act on a running job. Only the current lease owner may renew the lease or record the job's final outcome; a stale worker has no authority after the lease is reclaimed.
_Avoid_: process owner, shared worker identity

**Retry budget**:
The maximum number of execution attempts allowed for one job before it becomes dead-lettered. A job receives its own budget, defaulted from the queue configuration when it is submitted.
_Avoid_: global retry count

**Cancellation**:
An operator instruction that prevents a pending job from executing or asks a running job to stop cooperatively. Cancellation is not a guarantee that external side effects already started can be undone.
_Avoid_: hard stop, rollback
