import { useQuery } from "@tanstack/react-query";
import { fetchStats } from "../lib/api";

const STATUS_COLORS: Record<string, string> = {
  pending: "#f59e0b",
  running: "#3b82f6",
  success: "#22c55e",
  failed: "#ef4444",
  dead: "#6b7280",
};

const STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  running: "Running",
  success: "Success",
  failed: "Failed",
  dead: "Dead",
};

export default function StatsChart() {
  const { data: stats, isLoading, error } = useQuery({
    queryKey: ["stats"],
    queryFn: fetchStats,
    refetchInterval: 5000,
  });

  if (isLoading) return <p>Loading stats...</p>;
  if (error) return <p style={{ color: "red" }}>Stats error: {String(error)}</p>;
  if (!stats || Object.keys(stats).length === 0) return <p>No stats yet.</p>;

  const total = Object.values(stats).reduce((a, b) => a + b, 0);

  return (
    <div style={{ marginBottom: "1rem" }}>
      <h3 style={{ margin: "0 0 0.5rem 0" }}>Stats (last 5 min)</h3>
      <div style={{ display: "flex", gap: "0.75rem", flexWrap: "wrap" }}>
        {Object.entries(STATUS_COLORS).map(([status, color]) => {
          const count = stats[status] ?? 0;
          return (
            <div
              key={status}
              style={{
                flex: "1 0 100px",
                background: color,
                color: "#fff",
                borderRadius: 8,
                padding: "0.5rem 0.75rem",
                textAlign: "center",
                opacity: count === 0 ? 0.4 : 1,
              }}
            >
              <div style={{ fontSize: "1.5rem", fontWeight: 700 }}>{count}</div>
              <div style={{ fontSize: "0.75rem", textTransform: "uppercase", letterSpacing: "0.05em" }}>
                {STATUS_LABELS[status]}
              </div>
            </div>
          );
        })}
        <div
          style={{
            flex: "1 0 100px",
            background: "#1e293b",
            color: "#fff",
            borderRadius: 8,
            padding: "0.5rem 0.75rem",
            textAlign: "center",
          }}
        >
          <div style={{ fontSize: "1.5rem", fontWeight: 700 }}>{total}</div>
          <div style={{ fontSize: "0.75rem", textTransform: "uppercase", letterSpacing: "0.05em" }}>
            Total
          </div>
        </div>
      </div>
    </div>
  );
}