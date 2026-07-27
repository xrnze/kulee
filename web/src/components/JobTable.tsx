import { useMutation } from "@tanstack/react-query";
import { Job, retryJob, deleteJob, deleteAllDead } from "../lib/api";

interface Props {
  jobs: Job[];
  onAction: () => void;
}

export default function JobTable({ jobs, onAction }: Props) {
  const retry = useMutation({ mutationFn: retryJob, onSuccess: onAction });
  const remove = useMutation({ mutationFn: deleteJob, onSuccess: onAction });
  const purge = useMutation({ mutationFn: deleteAllDead, onSuccess: onAction });

  if (jobs.length === 0) {
    return (
      <div>
        <p>No jobs found.</p>
        {purge.isIdle && (
          <button onClick={() => purge.mutate()} disabled={purge.isPending}>
            Purge All Dead
          </button>
        )}
        {purge.data && <span>Deleted {purge.data.deleted} dead jobs</span>}
      </div>
    );
  }

  return (
    <div>
      <table style={{ width: "100%", borderCollapse: "collapse" }}>
        <thead>
          <tr style={{ textAlign: "left", borderBottom: "2px solid #ccc" }}>
            <th>ID</th>
            <th>Type</th>
            <th>Status</th>
            <th>Priority</th>
            <th>Attempts</th>
            <th>Error</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
            <tr key={job.id} style={{ borderBottom: "1px solid #eee" }}>
              <td>{job.id}</td>
              <td>{job.type}</td>
              <td>{job.status}</td>
              <td>{job.priority}</td>
              <td>{job.attempts}/{job.max_attempts}</td>
              <td style={{ maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis" }}>
                {job.last_error || "-"}
              </td>
              <td>
                {job.status === "dead" && (
                  <>
                    <button onClick={() => retry.mutate(job.id)} disabled={retry.isPending} style={{ marginRight: 4 }}>
                      Retry
                    </button>
                    <button onClick={() => remove.mutate(job.id)} disabled={remove.isPending}>
                      Delete
                    </button>
                  </>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={{ marginTop: "0.5rem" }}>
        <button onClick={() => purge.mutate()} disabled={purge.isPending}>
          Purge All Dead
        </button>
        {purge.data && <span style={{ marginLeft: "0.5rem" }}>Deleted {purge.data.deleted} dead jobs</span>}
      </div>
    </div>
  );
}