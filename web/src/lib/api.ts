export interface Job {
  id: number;
  type: string;
  status: string;
  priority: number;
  attempts: number;
  max_attempts: number;
  locked_by?: string | null;
  last_error?: string | null;
  created_at: string;
  updated_at: string;
}

export interface JobListResponse {
  jobs: Job[];
  next_cursor: number;
  has_more: boolean;
}

export interface Stats {
  [status: string]: number;
}

const BASE = "/api";

export async function listJobs(cursor = 0, limit = 20, status = ""): Promise<JobListResponse> {
  const params = new URLSearchParams({ cursor: String(cursor), limit: String(limit) });
  if (status) params.set("status", status);
  const res = await fetch(`${BASE}/jobs?${params}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function getJob(id: number): Promise<Job> {
  const res = await fetch(`${BASE}/jobs/${id}`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function enqueueJob(type: string, payload: unknown, priority = 1, maxAttempts = 5): Promise<Job> {
  const res = await fetch(`${BASE}/jobs`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ type, payload, priority, max_attempts: maxAttempts }),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function retryJob(id: number): Promise<Job> {
  const res = await fetch(`${BASE}/jobs/${id}/retry`, { method: "POST" });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function deleteJob(id: number): Promise<void> {
  const res = await fetch(`${BASE}/jobs/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(await res.text());
}

export async function deleteAllDead(): Promise<{ deleted: number }> {
  const res = await fetch(`${BASE}/jobs/dead`, { method: "DELETE" });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function fetchStats(): Promise<Stats> {
  const res = await fetch(`${BASE}/stats`);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}