"use client";

import { useFormStatus } from "react-dom";

export function PendingButton({
  children,
  className,
  pendingLabel = "Working…",
  disabled = false,
}: {
  children: React.ReactNode;
  className?: string;
  pendingLabel?: string;
  disabled?: boolean;
}) {
  const { pending } = useFormStatus();
  const blocked = pending || disabled;
  return (
    <button className={className} type="submit" disabled={blocked} aria-busy={pending}>
      {pending ? pendingLabel : children}
    </button>
  );
}
