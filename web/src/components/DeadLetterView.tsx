import { useQuery, useMutation } from "@tanstack/react-query";
import { listJobs, retryJob, deleteJob, deleteAllDead } from "../lib/api";

export default function DeadLetterView() {
  const {
    data,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["jobs", "dead"],
    queryFn: () => listJobs(0, 50, "dead"),
    refetchInterval: 5000,
  });

  const retry = useMutation({ mutationFn: retryJob, onSuccess: () => refetch() });
  const remove = useMutation({ mutationFn: deleteJob, onSuccess: () => refetch() });
  const purge = useMutation({ mutationFn: deleteAllDead, onSuccess: () => refetch() });

  const jobs = data?.jobs ?? [];

  return (
    <div style={{ marginTop: "1.5rem" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "0.75rem", marginBottom: "0.5rem" }}>
        <h3 style={{ margin: 0 }}>Dead Letter Queue</h3>
        <button
          onClick={() => purge.mutate()}
          disabled={purge.isPending || jobs.length === 0}
          style={{
            background: "#dc2626",
            color: "#fff",
            border: "none",
            borderRadius: 4,
            padding: "0.25rem 0.75rem",
            cursor: jobs.length === 0 ? "default" : "pointer",
            fontSize: "0.8rem",
          }}
        >
          {purge.isPending ? "Deleting..." : "Delete All Dead"}
        </button>
        {purge.data && (
          <span style={{ fontSize: "0.8rem", color: "#666" }}>
            Deleted {purge.data.deleted} dead job{purge.data.deleted !== 1 ? "s" : ""}
          </span>
        )}
      </div>

      {isLoading && <p>Loading dead jobs...</p>}
      {error && <p style={{ color: "red" }}>Error: {String(error)}</p>}

      {!isLoading && jobs.length === 0 && <p>No dead-lettered jobs.</p>}

      {jobs.length > 0 && (
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "0.9rem" }}>
          <thead>
            <tr style={{ textAlign: "left", borderBottom: "2px solid #ccc" }}>
              <th style={{ padding: "0.4rem 0.5rem" }}>ID</th>
              <th style={{ padding: "0.4rem 0.5rem" }}>Type</th>
              <th style={{ padding: "0.4rem 0.5rem" }}>Last Error</th>
              <th style={{ padding: "0.4rem 0.5rem" }}>Created</th>
              <th style={{ padding: "0.4rem 0.5rem" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.id} style={{ borderBottom: "1px solid #eee" }}>
                <td style={{ padding: "0.4rem 0.5rem" }}>{job.id}</td>
                <td style={{ padding: "0.4rem 0.5rem" }}>{job.type}</td>
                <td
                  style={{
                    padding: "0.4rem 0.5rem",
                    maxWidth: 300,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    color: "#dc2626",
                  }}
                >
                  {job.last_error || "-"}
                </td>
                <td style={{ padding: "0.4rem 0.5rem", whiteSpace: "nowrap" }}>
                  {new Date(job.created_at).toLocaleString()}
                </td>
                <td style={{ padding: "0.4rem 0.5rem" }}>
                  <button
                    onClick={() => retry.mutate(job.id)}
                    disabled={retry.isPending}
                    style={{ marginRight: "0.4rem", fontSize: "0.8rem" }}
                  >
                    Retry
                  </button>
                  <button
                    onClick={() => remove.mutate(job.id)}
                    disabled={remove.isPending}
                    style={{ fontSize: "0.8rem" }}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}