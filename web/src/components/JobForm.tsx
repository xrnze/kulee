import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { enqueueJob } from "../lib/api";

interface Props {
  onEnqueued: () => void;
}

const fieldClass =
  "h-11 w-full rounded-none border-2 border-black bg-white px-3 font-mono text-sm text-black placeholder:text-neutral-600 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black";

export default function JobForm({ onEnqueued }: Props) {
  const [type, setType] = useState("send_email");
  const [payload, setPayload] = useState('{"to":"user@example.com","subject":"Hello","body":"Test"}');

  const mutation = useMutation({
    mutationFn: () => enqueueJob(type, JSON.parse(payload)),
    onSuccess: onEnqueued,
  });

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        mutation.mutate();
      }}
      className="grid gap-4 lg:grid-cols-[14rem_minmax(0,1fr)_auto] lg:items-end"
    >
      <label className="grid gap-2 text-sm font-bold">
        Job type
        <select
          value={type}
          onChange={(event) => {
            setType(event.target.value);
            mutation.reset();
          }}
          className={fieldClass}
        >
          <option value="send_email">send_email</option>
          <option value="webhook_delivery">webhook_delivery</option>
          <option value="generate_report">generate_report</option>
        </select>
      </label>

      <label className="grid min-w-0 gap-2 text-sm font-bold">
        JSON payload
        <input
          type="text"
          value={payload}
          onChange={(event) => {
            setPayload(event.target.value);
            mutation.reset();
          }}
          spellCheck={false}
          className={fieldClass}
        />
      </label>

      <button
        type="submit"
        disabled={mutation.isPending}
        className="h-11 border-2 border-black bg-black px-6 text-sm font-black text-white hover:bg-white hover:text-black focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-not-allowed disabled:bg-neutral-400 disabled:text-neutral-800"
      >
        {mutation.isPending ? "ENQUEUING" : "ENQUEUE"}
      </button>

      {mutation.isError && (
        <p role="alert" className="font-mono text-sm font-bold lg:col-span-3">
          ERROR: {String(mutation.error)}
        </p>
      )}
    </form>
  );
}
