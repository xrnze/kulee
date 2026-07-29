import { useMutation } from "@tanstack/react-query";
import { Job, retryJob, deleteJob, deleteAllDead } from "../lib/api";

interface Props {
  jobs: Job[];
  status: string;
  isLoading: boolean;
  error: Error | null;
  onStatusChange: (status: string) => void;
  onAction: () => void;
}

const FILTERS = [
  { label: "All", value: "" },
  { label: "Pending", value: "pending" },
  { label: "Running", value: "running" },
  { label: "Success", value: "success" },
  { label: "Failed", value: "failed" },
  { label: "Dead", value: "dead" },
];

const STATUS_CLASSES: Record<string, string> = {
  pending: "border-dashed bg-white text-black",
  running: "bg-black text-white",
  success: "bg-neutral-200 text-black",
  failed: "bg-neutral-700 text-white",
  dead: "bg-neutral-400 text-black",
};

const actionClass =
  "h-9 border-2 border-black bg-white px-3 text-xs font-black hover:bg-black hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-not-allowed disabled:bg-neutral-300 disabled:text-neutral-600";

export default function JobTable({
  jobs,
  status,
  isLoading,
  error,
  onStatusChange,
  onAction,
}: Props) {
  const retry = useMutation({ mutationFn: retryJob, onSuccess: onAction });
  const remove = useMutation({ mutationFn: deleteJob, onSuccess: onAction });
  const purge = useMutation({ mutationFn: deleteAllDead, onSuccess: onAction });
  const actionError = retry.error ?? remove.error ?? purge.error;

  return (
    <div>
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap" role="group" aria-label="Filter jobs by status">
          {FILTERS.map((filter) => {
            const active = status === filter.value;
            return (
              <button
                key={filter.label}
                type="button"
                aria-pressed={active}
                onClick={() => onStatusChange(filter.value)}
                className={`h-10 border-2 border-black px-3 text-xs font-black first:ml-0 [&:not(:first-child)]:-ml-0.5 focus-visible:relative focus-visible:z-10 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black ${
                  active ? "relative z-10 bg-black text-white" : "bg-white text-black hover:bg-neutral-200"
                }`}
              >
                {filter.label.toUpperCase()}
              </button>
            );
          })}
        </div>

        {status === "dead" && (
          <button
            type="button"
            disabled={purge.isPending || jobs.length === 0}
            onClick={() => {
              if (window.confirm("Delete all dead jobs? This cannot be undone.")) {
                purge.mutate();
              }
            }}
            className={actionClass}
          >
            {purge.isPending ? "DELETING" : "DELETE ALL DEAD"}
          </button>
        )}
      </div>

      {purge.data && (
        <p aria-live="polite" className="mb-3 font-mono text-sm font-bold">
          DELETED {purge.data.deleted} DEAD JOB{purge.data.deleted === 1 ? "" : "S"}
        </p>
      )}

      {actionError && (
        <p role="alert" className="mb-3 border-2 border-black p-3 font-mono text-sm font-bold">
          ACTION ERROR: {String(actionError)}
        </p>
      )}

      {error ? (
        <p role="alert" className="border-2 border-black p-4 font-mono text-sm font-bold">
          JOBS ERROR: {String(error)}
        </p>
      ) : (
        <div className="overflow-x-auto border-2 border-black">
          <table className="w-full min-w-[70rem] border-collapse text-left text-sm">
            <caption className="sr-only">Jobs in the queue</caption>
            <thead className="bg-black text-white">
              <tr>
                <th scope="col" className="sticky left-0 z-20 w-20 min-w-20 bg-black px-3 py-3 font-mono text-xs">
                  ID
                </th>
                <th
                  scope="col"
                  className="sticky left-20 z-20 w-32 min-w-32 bg-black px-3 py-3 font-mono text-xs"
                >
                  STATUS
                </th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">TYPE</th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">PRIORITY</th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">ATTEMPTS</th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">ERROR</th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">CREATED</th>
                <th scope="col" className="px-3 py-3 font-mono text-xs">ACTIONS</th>
              </tr>
            </thead>
            <tbody>
              {isLoading &&
                Array.from({ length: 5 }, (_, index) => (
                  <tr key={index} aria-hidden="true" className="border-b border-black last:border-b-0">
                    <td colSpan={8} className="p-3">
                      <div className="h-5 bg-neutral-200 motion-safe:animate-pulse" />
                    </td>
                  </tr>
                ))}

              {!isLoading && jobs.length === 0 && (
                <tr>
                  <td colSpan={8} className="p-8 text-center font-mono text-sm font-bold">
                    NO JOBS MATCH THIS FILTER
                  </td>
                </tr>
              )}

              {!isLoading &&
                jobs.map((job) => (
                  <tr key={job.id} className="group border-b border-black last:border-b-0 hover:bg-neutral-100">
                    <td className="sticky left-0 z-10 bg-white px-3 py-3 font-mono font-bold group-hover:bg-neutral-100">
                      #{job.id}
                    </td>
                    <td className="sticky left-20 z-10 bg-white px-3 py-3 group-hover:bg-neutral-100">
                      <span
                        className={`inline-flex border-2 border-black px-2 py-1 font-mono text-xs font-black uppercase ${
                          STATUS_CLASSES[job.status] ?? "bg-white text-black"
                        }`}
                      >
                        {job.status}
                      </span>
                    </td>
                    <td className="px-3 py-3 font-mono">{job.type}</td>
                    <td className="px-3 py-3 font-mono">{job.priority}</td>
                    <td className="px-3 py-3 font-mono">
                      {job.attempts}/{job.max_attempts}
                    </td>
                    <td className="max-w-xs px-3 py-3 font-mono">
                      <span className="block truncate" title={job.last_error ?? undefined}>
                        {job.last_error || "-"}
                      </span>
                    </td>
                    <td className="whitespace-nowrap px-3 py-3 font-mono text-xs">
                      <time dateTime={job.created_at}>{new Date(job.created_at).toLocaleString()}</time>
                    </td>
                    <td className="px-3 py-3">
                      {job.status === "dead" ? (
                        <div className="flex gap-2">
                          <button
                            type="button"
                            onClick={() => retry.mutate(job.id)}
                            disabled={retry.isPending}
                            className={actionClass}
                          >
                            RETRY
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              if (window.confirm(`Delete dead job #${job.id}? This cannot be undone.`)) {
                                remove.mutate(job.id);
                              }
                            }}
                            disabled={remove.isPending}
                            className={actionClass}
                          >
                            DELETE
                          </button>
                        </div>
                      ) : (
                        <span aria-hidden="true" className="font-mono text-neutral-500">
                          -
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
