import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { enqueueJob } from "../lib/api";

interface Props {
  onEnqueued: () => void;
}

export default function JobForm({ onEnqueued }: Props) {
  const [type, setType] = useState("send_email");
  const [payload, setPayload] = useState('{"to":"user@example.com","subject":"Hello","body":"Test"}');

  const mutation = useMutation({
    mutationFn: () => enqueueJob(type, JSON.parse(payload)),
    onSuccess: onEnqueued,
  });

  return (
    <div style={{ display: "flex", gap: "0.5rem", alignItems: "center", flexWrap: "wrap" }}>
      <select value={type} onChange={(e) => setType(e.target.value)}>
        <option value="send_email">send_email</option>
        <option value="webhook_delivery">webhook_delivery</option>
        <option value="generate_report">generate_report</option>
      </select>
      <input
        type="text"
        value={payload}
        onChange={(e) => setPayload(e.target.value)}
        style={{ width: 300, fontFamily: "monospace" }}
      />
      <button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
        {mutation.isPending ? "Enqueuing..." : "Enqueue"}
      </button>
      {mutation.isError && <span style={{ color: "red" }}>Error: {String(mutation.error)}</span>}
    </div>
  );
}