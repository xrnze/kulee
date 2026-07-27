import { useQuery } from "@tanstack/react-query";
import { listJobs } from "./lib/api";
import JobTable from "./components/JobTable";
import JobForm from "./components/JobForm";
import StatsChart from "./components/StatsChart";
import DeadLetterView from "./components/DeadLetterView";

export default function App() {
  const {
    data: jobsData,
    isLoading: jobsLoading,
    refetch: refetchJobs,
  } = useQuery({
    queryKey: ["jobs"],
    queryFn: () => listJobs(0, 20, ""),
    refetchInterval: 5000,
  });

  return (
    <div style={{ maxWidth: 960, margin: "0 auto", padding: "1rem", fontFamily: "system-ui, sans-serif" }}>
      <h1>Kulee Job Queue</h1>

      <div style={{ display: "flex", gap: "1rem", alignItems: "center", marginBottom: "1rem" }}>
        <JobForm onEnqueued={() => refetchJobs()} />
        <button onClick={() => refetchJobs()}>Refresh</button>
      </div>

      <StatsChart />

      {jobsLoading ? <p>Loading...</p> : <JobTable jobs={jobsData?.jobs ?? []} onAction={refetchJobs} />}

      <DeadLetterView />
    </div>
  );
}