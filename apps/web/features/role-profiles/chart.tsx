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
    <section className="role-workspace" aria-labelledby="organogram-workspace-heading">
      <header className="role-workspace-intro">
        <h2 id="organogram-workspace-heading">Organogram workspace</h2>
        <p>Create a chart, import a file, or build positions online before you open a role profile.</p>
      </header>

      <div className="role-workspace-grid">
        <article className="role-card" aria-labelledby="chart-management-heading">
          <header className="role-card-head">
            <h3 id="chart-management-heading">Charts and uploads</h3>
            <p>Name a chart, then upload a spreadsheet, import the template, or switch to the online builder.</p>
          </header>

          {!charts.ok ? <StatusMessage tone="error">{charts.error}</StatusMessage> : null}
          {charts.ok && charts.value.length === 0 ? (
            <EmptyState>No organogram is recorded yet. Name a chart, then build it, upload a file, or import the template.</EmptyState>
          ) : null}

          {charts.ok && charts.value.length > 0 ? (
            <ul className="role-chart-list">
              {charts.value.map((item) => (
                <li key={item.id}>
                  <Link href={chartHref(item.id, mode)} aria-current={chart?.id === item.id ? "true" : undefined}>
                    <strong>{item.name}</strong>
                    <span className={item.status === "published" ? "status-pill" : "role-status-draft"}>
                      {item.status === "published" ? "Published" : "Draft"}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          ) : null}

          <form action={createOrganogramForm} className="framework-form role-card-form">
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

          {mode === "upload" ? (
            <div className="role-card-block">
              <h4>Upload a spreadsheet or chart image</h4>
              {chart ? <OrganogramUpload organogramId={chart.id} /> : <p className="portal-note">Save a chart name before uploading a file.</p>}
            </div>
          ) : null}

          {mode === "template" ? (
            <div className="role-card-block chart-template">
              <h4>Import the template</h4>
              <p className="portal-note">
                The template has title, reports_to, and department columns and no sample positions. A department name must already exist in the
                directory.
              </p>
              {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
              <a href="/admin/role-profiles/organogram-template">Download template</a>
              {chart ? <OrganogramUpload organogramId={chart.id} /> : <p className="portal-note">Save a chart name before importing the completed template.</p>}
            </div>
          ) : null}

          {files && !files.ok ? <StatusMessage tone="error">{files.error}</StatusMessage> : null}
          {files?.ok && files.value.length > 0 ? (
            <div className="role-card-block">
              <h4>Uploaded files</h4>
              <ul className="role-file-list">
                {files.value.map((file) => (
                  <li key={file.id}>
                    <a href={`/admin/role-profiles/charts/${encodeURIComponent(file.organogramId)}/files/${encodeURIComponent(file.id)}`}>
                      {file.fileName}
                    </a>
                    <span>
                      {file.status}: {file.detail}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </article>

        {chart ? (
          <article className="role-card role-card-status" aria-labelledby="chart-status-heading">
            <header className="role-card-head">
              <h3 id="chart-status-heading">Chart status</h3>
              <p>Publishing locks the chart for the organisation directory workflow.</p>
            </header>
            <p className="role-status-line">
              <strong>{chart.name}</strong>
              <span className={chart.status === "published" ? "status-pill" : "role-status-draft"}>
                {chart.status === "published" ? "Published" : "Draft"}
              </span>
            </p>
            <p className="portal-note">
              {chart.status === "published"
                ? "This chart is published. Positions can still be reviewed from the builder."
                : "This chart is still a draft. Publish it when the positions are ready."}
            </p>
            {positions?.ok ? (
              <p className="role-status-metric">
                <strong>{positions.value.length}</strong>
                <span>{positions.value.length === 1 ? "position recorded" : "positions recorded"}</span>
              </p>
            ) : null}
            {positions && !positions.ok ? <StatusMessage tone="error">{positions.error}</StatusMessage> : null}
          </article>
        ) : null}
      </div>

      {mode === "build" && chart && positions?.ok ? (
        <article className="role-card role-card-wide" aria-labelledby="builder-heading">
          <header className="role-card-head">
            <h3 id="builder-heading">Online builder and positions</h3>
            <p>Search the hierarchy, add positions, and publish the draft chart when it is ready.</p>
          </header>
          <OrganogramEditor chart={chart} positions={positions.value} departments={departments} people={people} />
        </article>
      ) : null}
      {mode === "build" && !chart ? <p className="portal-note">Save a chart name before building positions online.</p> : null}
      {mode === "build" && positions && !positions.ok ? <StatusMessage tone="error">{positions.error}</StatusMessage> : null}
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
