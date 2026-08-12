import { useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { listJobs } from "./lib/api";
import JobTable from "./components/JobTable";
import JobForm from "./components/JobForm";
import StatsChart from "./components/StatsChart";

const PAGE_SIZE = 10;

export default function App() {
  const [status, setStatus] = useState("");
  const [cursor, setCursor] = useState(0);
  const [history, setHistory] = useState<number[]>([]);
  const queryClient = useQueryClient();

  const {
    data: jobsData,
    isLoading: jobsLoading,
    isFetching: jobsFetching,
    error: jobsError,
    refetch: refetchJobs,
  } = useQuery({
    queryKey: ["jobs", status, cursor],
    queryFn: () => listJobs(cursor, PAGE_SIZE, status),
    refetchInterval: 5000,
  });

  const invalidateStats = () => queryClient.invalidateQueries({ queryKey: ["stats"] });

  const changeStatus = (next: string) => {
    setStatus(next);
    setCursor(0);
    setHistory([]);
  };

  const goNext = () => {
    const next = jobsData?.next_cursor ?? 0;
    if (!next) return;
    setHistory((prev) => [...prev, cursor]);
    setCursor(next);
  };

  const goPrev = () => {
    const last = history[history.length - 1] ?? 0;
    setHistory(history.slice(0, -1));
    setCursor(last);
  };

  const [toast, setToast] = useState<string | null>(null);
  const toastTimer = useRef<number | null>(null);
  const notify = (msg: string) => {
    setToast(msg);
    if (toastTimer.current) window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(null), 4000);
  };

  return (
    <div className="min-h-screen bg-neutral-200 px-3 py-3 font-sans text-black sm:px-6 sm:py-6">
      <main className="mx-auto max-w-7xl border-2 border-black bg-white">
        <header className="flex flex-col gap-5 p-5 sm:flex-row sm:items-end sm:justify-between sm:p-7">
          <div>
            <h1 className="text-3xl font-black leading-none sm:text-4xl">KULEE</h1>
            <p className="mt-2 font-mono text-sm font-bold">JOB QUEUE CONTROL</p>
          </div>

          <div className="flex items-center gap-3">
            <span className="font-mono text-xs font-bold" aria-live="polite">
              {jobsFetching ? "SYNCING" : "AUTO REFRESH / 5S"}
            </span>
            <button
              type="button"
              onClick={() => void refetchJobs()}
              disabled={jobsFetching}
              className="h-11 border-2 border-black bg-white px-4 text-sm font-bold hover:bg-black hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-not-allowed disabled:bg-neutral-300 disabled:text-neutral-600"
            >
              REFRESH
            </button>
          </div>
        </header>

        <section aria-labelledby="dispatch-heading" className="border-t-2 border-black p-5 sm:p-7">
          <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
            <h2 id="dispatch-heading" className="text-xl font-black">
              Dispatch job
            </h2>
            <p className="font-mono text-xs">POST /api/jobs</p>
          </div>
          <JobForm onEnqueued={(job) => { void refetchJobs(); invalidateStats(); notify(`ENQUEUED #${job.id}`); }} />
        </section>

        <StatsChart />

        <section aria-labelledby="ledger-heading" className="border-t-2 border-black p-5 sm:p-7">
          <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
            <h2 id="ledger-heading" className="text-xl font-black">
              Job ledger
            </h2>
            <p className="font-mono text-xs">{jobsData?.jobs.length ?? 0} RECORDS</p>
          </div>
          <JobTable
            jobs={jobsData?.jobs ?? []}
            status={status}
            isLoading={jobsLoading}
            error={jobsError}
            hasMore={jobsData?.has_more ?? false}
            canPrev={history.length > 0}
            onStatusChange={changeStatus}
            onNext={goNext}
            onPrev={goPrev}
            onAction={() => { void refetchJobs(); invalidateStats(); }}
            notify={notify}
          />
        </section>
      </main>
      {toast && (
        <div
          role="status"
          aria-live="polite"
          className="fixed bottom-4 right-4 z-40 border-2 border-black bg-white px-4 py-3 font-mono text-sm font-bold"
        >
          {toast}
        </div>
      )}
    </div>
  );
}
