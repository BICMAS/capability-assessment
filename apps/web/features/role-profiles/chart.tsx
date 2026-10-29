import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { Department, DirectoryPerson, Organogram, OrganogramFile, Position } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

import { createOrganogramForm } from "./chart-actions";
import { OrganogramEditor } from "./organogram-editor";
import { OrganogramUpload } from "./organogram-upload";

export function OrganogramStage({
  charts,
  chart,
  positions,
  files,
  departments,
  people,
  entry,
}: {
  charts: Settled<Organogram[]>;
  chart: Organogram | null;
  positions: Settled<Position[]> | null;
  files: Settled<OrganogramFile[]> | null;
  departments: Department[];
  people: DirectoryPerson[];
  entry: string;
}) {
  const mode = entry === "upload" || entry === "template" ? entry : "build";
  return (
    <section className="directory-actions" aria-labelledby="organogram-stage-heading">
      <h2 id="organogram-stage-heading">Organogram</h2>
      {!charts.ok ? <StatusMessage tone="error">{charts.error}</StatusMessage> : null}
      {charts.ok && charts.value.length === 0 ? <EmptyState>No organogram is recorded yet. Name a chart, then build it, upload a file, or import the template.</EmptyState> : null}
      {charts.ok && charts.value.length > 0 ? (
        <ul className="organogram">
          {charts.value.map((item) => (
            <li key={item.id}>
              <Link href={`/admin/role-profiles?chart=${encodeURIComponent(item.id)}&entry=${mode}`} aria-current={chart?.id === item.id ? "true" : undefined}>
                <strong>{item.name}</strong>
              </Link>
              <span>{item.status === "published" ? "Published" : "Draft"}</span>
            </li>
          ))}
        </ul>
      ) : null}
      <form action={createOrganogramForm} className="framework-form">
        <label>
          Chart name
          <input name="name" required />
        </label>
        <PendingButton className="primary">Save organogram</PendingButton>
      </form>
      <div className="chart-modes" role="group" aria-label="Organogram entry">
        <Link href={chartHref(chart?.id, "upload")} aria-current={mode === "upload" ? "true" : undefined}>
          Upload file
        </Link>
        <Link href={chartHref(chart?.id, "build")} aria-current={mode === "build" ? "true" : undefined}>
          Build online
        </Link>
        <Link href={chartHref(chart?.id, "template")} aria-current={mode === "template" ? "true" : undefined}>
          Use template
        </Link>
      </div>
      {mode === "upload" ? chart ? <OrganogramUpload organogramId={chart.id} /> : <p>Save a chart name before uploading a file.</p> : null}
      {mode === "build" && chart && positions?.ok && departments ? (
        <OrganogramEditor chart={chart} positions={positions.value} departments={departments} people={people} />
      ) : null}
      {mode === "build" && positions && !positions.ok ? <StatusMessage tone="error">{positions.error}</StatusMessage> : null}
      {mode === "template" ? (
        <div className="chart-template">
          <p className="portal-note">The template has title, reports_to, and department columns and no sample positions. A department name must already exist in the directory.</p>
          {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
          <a href="/admin/role-profiles/organogram-template">Download template</a>
          {chart ? <OrganogramUpload organogramId={chart.id} /> : <p>Save a chart name before importing the completed template.</p>}
        </div>
      ) : null}
      {files && !files.ok ? <StatusMessage tone="error">{files.error}</StatusMessage> : null}
      {files?.ok && files.value.length > 0 ? (
        <ul className="organogram">
          {files.value.map((file) => (
            <li key={file.id}>
              <a href={`/admin/role-profiles/charts/${encodeURIComponent(file.organogramId)}/files/${encodeURIComponent(file.id)}`}>{file.fileName}</a>
              <span>
                {file.status}: {file.detail}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function chartHref(chartId: string | undefined, entry: string): string {
  const query = new URLSearchParams({ entry });
  if (chartId) {
    query.set("chart", chartId);
  }
  return `/admin/role-profiles?${query.toString()}`;
}
