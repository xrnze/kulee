import { useQuery } from "@tanstack/react-query";
import { fetchStats } from "../lib/api";

const STATUSES = ["pending", "running", "success", "failed", "dead"] as const;

export default function StatsChart() {
  const { data: stats, isLoading, isFetching, error } = useQuery({
    queryKey: ["stats"],
    queryFn: fetchStats,
    refetchInterval: () => {
      if (typeof document !== "undefined" && document.hidden) return false;
      return 5000;
    },
  });

  const total = stats ? Object.values(stats).reduce((sum, count) => sum + count, 0) : 0;
  const metrics = [
    { label: "Total", count: total },
    ...STATUSES.map((status) => ({ label: status, count: stats?.[status] ?? 0 })),
  ];

  return (
    <section aria-labelledby="metrics-heading" className="border-t-2 border-black p-5 sm:p-7">
      <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="metrics-heading" className="text-xl font-black">
          Queue metrics
        </h2>
        <div className="flex items-center gap-3">
          <p className="font-mono text-xs">LAST 5 MINUTES</p>
          {isFetching && !isLoading && (
            <span className="font-mono text-xs motion-safe:animate-pulse">SYNCING</span>
          )}
        </div>
      </div>

      {error && stats && (
        <p role="alert" className="mb-3 border-2 border-black p-3 font-mono text-xs font-bold">
          STATS ERROR: {String(error)}
        </p>
      )}

      <div
        aria-busy={isLoading}
        className="grid grid-cols-2 gap-0.5 border-2 border-black bg-black sm:grid-cols-3 lg:grid-cols-6"
      >
        {metrics.map(({ label, count }, index) => (
          <div
            key={label}
            className={`min-h-24 p-4 ${index === 0 ? "bg-black text-white" : "bg-white text-black"}`}
          >
            <p className={`font-mono text-3xl font-black ${isLoading ? "motion-safe:animate-pulse" : ""}`}>
              {isLoading ? "--" : count}
            </p>
            <p className="mt-2 text-xs font-bold uppercase">{label}</p>
          </div>
        ))}
      </div>

      {error && !stats && (
        <p role="alert" className="mt-3 border-2 border-black p-3 font-mono text-sm font-bold">
          STATS ERROR: {String(error)}
        </p>
      )}
    </section>
  );
}