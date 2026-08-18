import { useEffect, useId, useRef } from "react";

interface Props {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  busy?: boolean;
  error?: string | null;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  busy = false,
  error = null,
  onConfirm,
  onCancel,
}: Props) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  const descriptionId = useId();
  const previouslyFocused = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      previouslyFocused.current = document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;
      dialog.showModal();
      confirmRef.current?.focus();
    } else if (!open && dialog.open) {
      dialog.close();
      previouslyFocused.current?.focus();
      previouslyFocused.current = null;
    }
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      aria-describedby={descriptionId}
      className="kulee-dialog w-[calc(100%-1.5rem)] max-w-lg bg-white p-0 text-black"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onCancel();
      }}
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !busy) onCancel();
      }}
    >
      <div className="border-b-2 border-black bg-black px-5 py-4 text-white sm:px-7">
        <p className="font-mono text-xs font-bold">CONFIRM ACTION</p>
        <h2 id={titleId} className="mt-1 text-xl font-black">
          {title}
        </h2>
      </div>
      <div className="p-5 sm:p-7">
        <p id={descriptionId} className="max-w-prose font-mono text-sm leading-6">
          {description}
        </p>
        {error && (
          <p role="alert" className="mt-4 border-2 border-black bg-neutral-200 p-3 font-mono text-sm font-bold">
            DELETE ERROR: {error}
          </p>
        )}
        <div className="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={onCancel}
            disabled={busy}
            className="h-11 border-2 border-black bg-white px-4 text-sm font-black hover:bg-black hover:text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-not-allowed disabled:bg-neutral-300 disabled:text-neutral-600"
          >
            CANCEL
          </button>
          <button
            ref={confirmRef}
            type="button"
            onClick={onConfirm}
            disabled={busy}
            className="h-11 border-2 border-black bg-black px-6 text-sm font-black text-white hover:bg-white hover:text-black focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white disabled:cursor-not-allowed disabled:bg-neutral-700 disabled:text-neutral-300"
          >
            {busy ? "DELETING" : confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  );
}
