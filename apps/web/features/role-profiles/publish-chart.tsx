"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { publishOrganogramAction } from "./chart-actions";

export function PublishChartButton({
  organogramId,
  status,
  positionCount,
}: {
  organogramId: string;
  status: string;
  positionCount: number;
}) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  if (status === "published") {
    return <p className="portal-note">This chart is already published.</p>;
  }

  function start() {
    setMessage("");
    if (positionCount < 1) {
      setMessage("Add at least one position before publishing this chart.");
      return;
    }
    setOpen(true);
  }

  async function confirm() {
    setPending(true);
    const result = await publishOrganogramAction(organogramId);
    setPending(false);
    if (!result.ok) {
      setMessage(result.message);
      setOpen(false);
      return;
    }
    setOpen(false);
    router.refresh();
  }

  return (
    <div className="organo-publish">
      <button type="button" className="primary" onClick={start} disabled={pending}>
        Publish chart
      </button>
      {message ? (
        <p role="alert" className="status-message is-error">
          {message}
        </p>
      ) : null}
      {open ? (
        <div className="chart-confirm" role="dialog" aria-modal="true" aria-labelledby="publish-chart-title">
          <h3 id="publish-chart-title">Publish this organogram?</h3>
          <p>
            Publishing marks the chart as published for the organisation. It currently has {positionCount}{" "}
            {positionCount === 1 ? "position" : "positions"}.
          </p>
          <button type="button" onClick={() => setOpen(false)} disabled={pending}>
            Cancel
          </button>
          <button type="button" className="primary" onClick={() => void confirm()} disabled={pending}>
            {pending ? "Publishing…" : "Confirm publish"}
          </button>
        </div>
      ) : null}
    </div>
  );
}
