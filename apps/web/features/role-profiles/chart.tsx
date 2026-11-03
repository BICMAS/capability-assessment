import { FileSpreadsheet, Network, Upload } from "lucide-react";
import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { Department, DirectoryPerson, Organogram, OrganogramFile, Position } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

import { createOrganogramForm } from "./chart-actions";
import { ChartPreview } from "./chart-preview";
import { OrganogramEditor } from "./organogram-editor";
import { OrganogramUpload } from "./organogram-upload";
import { PublishChartButton } from "./publish-chart";

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
  const positionList = positions?.ok ? positions.value : [];
  const departmentCount = countDepartments(positionList);
  const lastActivity = latestActivity(chart, positionList, files?.ok ? files.value : []);

  return (
    <section className="organo-onboard" aria-labelledby="organo-onboard-heading">
      <header className="role-workspace-intro">
        <h2 id="organo-onboard-heading">Organogram onboarding</h2>
        <p>Name a chart, import a structure, or build positions online before you open a role profile.</p>
      </header>

      <article className="role-card" aria-labelledby="chart-select-heading">
        <header className="role-card-head">
          <h3 id="chart-select-heading">Choose or create a chart</h3>
          <p>Every upload and position belongs to a named organogram.</p>
        </header>
        {!charts.ok ? <StatusMessage tone="error">{charts.error}</StatusMessage> : null}
        {charts.ok && charts.value.length === 0 ? (
          <EmptyState>No organogram is recorded yet. Save a draft chart name to begin.</EmptyState>
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
            <input name="name" required placeholder="For example, Group leadership chart" />
          </label>
          <PendingButton className="primary">Save draft</PendingButton>
        </form>
      </article>

      {chart ? (
        <section className="organo-overview" aria-labelledby="organo-overview-heading">
          <header className="role-card-head">
            <h3 id="organo-overview-heading">Organogram overview</h3>
            <p>Live counts from the selected chart and its recorded positions.</p>
          </header>
          <div className="organo-overview-grid">
            <div>
              <span>Organogram</span>
              <strong>{chart.name}</strong>
            </div>
            <div>
              <span>Status</span>
              <strong className={chart.status === "published" ? "status-pill" : "role-status-draft"}>
                {chart.status === "published" ? "Published" : "Draft"}
              </strong>
            </div>
            <div>
              <span>Created</span>
              <strong>{formatWhen(chart.createdAt)}</strong>
            </div>
            <div>
              <span>Last activity</span>
              <strong>{lastActivity ? formatWhen(lastActivity) : "Not recorded"}</strong>
            </div>
            <div>
              <span>Departments</span>
              <strong>{positions?.ok ? departmentCount : "—"}</strong>
            </div>
            <div>
              <span>Positions</span>
              <strong>{positions?.ok ? positionList.length : "—"}</strong>
            </div>
          </div>
          {positions && !positions.ok ? <StatusMessage tone="error">{positions.error}</StatusMessage> : null}
        </section>
      ) : null}

      <section className="organo-methods" aria-labelledby="organo-methods-heading">
        <header className="role-card-head">
          <h3 id="organo-methods-heading">Alternative methods</h3>
          <p>Choose how you want to structure this chart. Only the selected method expands below.</p>
        </header>
        <div className="organo-method-grid" role="group" aria-label="Organogram entry">
          <Link className="organo-method" href={chartHref(chart?.id, "upload")} aria-current={mode === "upload" ? "true" : undefined}>
            <span className="organo-method-icon" aria-hidden="true">
              <Upload size={20} />
            </span>
            <strong>Upload spreadsheet</strong>
            <span>Import PNG, JPG, PDF, XLS, or XLSX through the existing upload service.</span>
          </Link>
          <Link className="organo-method" href={chartHref(chart?.id, "build")} aria-current={mode === "build" ? "true" : undefined}>
            <span className="organo-method-icon" aria-hidden="true">
              <Network size={20} />
            </span>
            <strong>Build online</strong>
            <span>Add positions, reporting lines, and people directly in the chart builder.</span>
          </Link>
          <Link className="organo-method" href={chartHref(chart?.id, "template")} aria-current={mode === "template" ? "true" : undefined}>
            <span className="organo-method-icon" aria-hidden="true">
              <FileSpreadsheet size={20} />
            </span>
            <strong>Use template</strong>
            <span>Download the blank template, complete it, then import the finished file.</span>
          </Link>
        </div>
      </section>

      {mode === "upload" ? (
        <section className="role-card organo-import" aria-labelledby="organo-import-heading">
          <h3 id="organo-import-heading" className="app-clip">
            Import organogram
          </h3>
          {chart ? (
            <OrganogramUpload organogramId={chart.id} />
          ) : (
            <EmptyState>Save a draft chart name before uploading a file.</EmptyState>
          )}
        </section>
      ) : null}

      {mode === "template" ? (
        <section className="role-card chart-template" aria-labelledby="organo-template-heading">
          <header className="role-card-head">
            <h3 id="organo-template-heading">Use template</h3>
            <p>
              The template has title, reports_to, and department columns and no sample positions. A department name must already
              exist in the directory.
            </p>
          </header>
          {/* eslint-disable-next-line @next/next/no-html-link-for-pages */}
          <a className="learn-link directory-secondary" href="/admin/role-profiles/organogram-template">
            Download template
          </a>
          {chart ? <OrganogramUpload organogramId={chart.id} /> : <EmptyState>Save a draft chart name before importing the completed template.</EmptyState>}
        </section>
      ) : null}

      {files && !files.ok ? <StatusMessage tone="error">{files.error}</StatusMessage> : null}
      {files?.ok && files.value.length > 0 ? (
        <section className="role-card" aria-labelledby="organo-files-heading">
          <header className="role-card-head">
            <h3 id="organo-files-heading">Uploaded files</h3>
            <p>Files stored against this chart, including import results.</p>
          </header>
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
        </section>
      ) : null}

      {chart ? (
        <section className="role-card organo-actions" aria-labelledby="organo-actions-heading">
          <header className="role-card-head">
            <h3 id="organo-actions-heading">Chart actions</h3>
            <p>Save a new draft chart, preview the recorded hierarchy, or publish when positions are ready.</p>
          </header>
          <div className="organo-action-row">
            <form action={createOrganogramForm} className="framework-form">
              <label>
                New draft name
                <input name="name" required placeholder="Save another draft chart" />
              </label>
              <PendingButton className="primary">Save draft</PendingButton>
            </form>
            <div className="organo-action-stack">
              <Link className="learn-link directory-secondary" href={`${chartHref(chart.id, "build")}#organo-preview`}>
                Preview chart
              </Link>
              <PublishChartButton organogramId={chart.id} status={chart.status} positionCount={positionList.length} />
            </div>
          </div>
          <div id="organo-preview" className="organo-preview">
            <h4>Chart preview</h4>
            {positions?.ok ? <ChartPreview positions={positionList} departments={departments} /> : null}
            {positions && !positions.ok ? <StatusMessage tone="error">{positions.error}</StatusMessage> : null}
          </div>
        </section>
      ) : null}

      {mode === "build" && chart && positions?.ok ? (
        <article className="role-card role-card-wide" aria-labelledby="builder-heading">
          <header className="role-card-head">
            <h3 id="builder-heading">Organisation structure editor</h3>
            <p>Browse positions, edit the reporting chart, and save changes before you publish.</p>
          </header>
          <OrganogramEditor chart={chart} positions={positions.value} departments={departments} people={people} />
        </article>
      ) : null}
      {mode === "build" && !chart ? <EmptyState>Save a draft chart name before building positions online.</EmptyState> : null}
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

function countDepartments(positions: Position[]): number {
  return new Set(positions.map((position) => position.departmentId).filter((id) => id !== "")).size;
}

function latestActivity(chart: Organogram | null, positions: Position[], files: OrganogramFile[]): string | null {
  const stamps = [
    chart?.createdAt ?? "",
    ...positions.map((position) => position.createdAt),
    ...files.map((file) => file.createdAt),
  ].filter((value) => value !== "");
  if (stamps.length === 0) {
    return null;
  }
  return stamps.reduce((latest, value) => (Date.parse(value) > Date.parse(latest) ? value : latest));
}

function formatWhen(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: "UTC",
  }).format(new Date(parsed));
}
