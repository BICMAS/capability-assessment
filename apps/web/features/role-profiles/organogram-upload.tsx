"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { organogramUploadIssue } from "./upload-rules";

type ImportResult = {
  file: { id: string; status: string; detail: string; fileName: string };
  accepted: number;
  rejected: { row: number; message: string }[];
  headers: string[];
};

export function OrganogramUpload({ organogramId }: { organogramId: string }) {
  const router = useRouter();
  const [file, setFile] = useState<File | null>(null);
  const [progress, setProgress] = useState("");
  const [error, setError] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);
  const [titleColumn, setTitleColumn] = useState("");
  const [reportsToColumn, setReportsToColumn] = useState("");
  const [departmentColumn, setDepartmentColumn] = useState("");

  function choose(next: File | null) {
    setResult(null);
    setError("");
    if (!next) {
      setFile(null);
      return;
    }
    const issue = organogramUploadIssue(next);
    if (issue) {
      setFile(null);
      setError(issue);
      return;
    }
    setFile(next);
  }

  function send(selected: File) {
    setError("");
    setProgress("Uploading 0%");
    const body = new FormData();
    body.set("file", selected);
    if (titleColumn) {
      body.set("titleColumn", titleColumn);
    }
    if (reportsToColumn) {
      body.set("reportsToColumn", reportsToColumn);
    }
    if (departmentColumn) {
      body.set("departmentColumn", departmentColumn);
    }
    const request = new XMLHttpRequest();
    request.open("POST", `/admin/role-profiles/charts/${encodeURIComponent(organogramId)}/files`);
    request.upload.onprogress = (event) => {
      if (event.lengthComputable) {
        setProgress(`Uploading ${Math.round((event.loaded / event.total) * 100)}%`);
      }
    };
    request.upload.onload = () => setProgress("Processing the file.");
    request.onload = () => {
      setProgress("");
      if (request.status === 201) {
        try {
          const payload = JSON.parse(request.responseText) as ImportResult;
          setResult(payload);
          if (payload.accepted > 0) {
            router.refresh();
          }
        } catch {
          setError("The import response could not be read.");
        }
        return;
      }
      setError(request.responseText.trim() || "The file could not be processed.");
    };
    request.onerror = () => {
      setProgress("");
      setError("The upload was interrupted. Retry when the connection is available.");
    };
    request.send(body);
  }

  const needsMapping = result?.file.status === "rejected" && result.headers.length > 0;
  return (
    <div className="chart-upload">
      <p className="portal-note">PNG, JPG, and PDF files are stored for review. They are not turned into positions. XLSX files can import a title, a reporting line, and a department. XLS files must be saved as XLSX first.</p>
      <div
        className="chart-drop"
        onDragOver={(event) => event.preventDefault()}
        onDrop={(event) => {
          event.preventDefault();
          choose(event.dataTransfer.files[0] ?? null);
        }}
      >
        <p>Drag a file here or choose one.</p>
        <input
          aria-label="Organogram file"
          type="file"
          accept=".png,.jpg,.jpeg,.pdf,.xls,.xlsx,image/png,image/jpeg,application/pdf"
          onChange={(event) => choose(event.target.files?.[0] ?? null)}
        />
      </div>
      {file ? <p>Selected: {file.name}</p> : null}
      {progress ? <p role="status">{progress}</p> : null}
      {error ? (
        <p role="alert" className="status-message is-error">
          {error}
        </p>
      ) : null}
      {needsMapping ? (
        <fieldset className="framework-form">
          <legend>Map the spreadsheet columns</legend>
          <ColumnSelect label="Title column" headers={result.headers} value={titleColumn} onChange={setTitleColumn} />
          <ColumnSelect label="Reports-to column" headers={result.headers} value={reportsToColumn} onChange={setReportsToColumn} />
          <ColumnSelect label="Department column" headers={result.headers} value={departmentColumn} onChange={setDepartmentColumn} />
        </fieldset>
      ) : null}
      <button type="button" className="primary" disabled={!file || progress !== ""} onClick={() => file && send(file)}>
        Upload file
      </button>
      {error && file ? (
        <button type="button" onClick={() => send(file)}>
          Retry upload
        </button>
      ) : null}
      {result ? (
        <div>
          <p role="status">
            {result.file.fileName}: {result.file.detail}
          </p>
          <p>
            {result.accepted} accepted, {result.rejected.length} rejected.
          </p>
          {result.rejected.length > 0 ? (
            <ul>
              {result.rejected.map((row) => (
                <li key={`${row.row}-${row.message}`}>
                  Row {row.row}: {row.message}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function ColumnSelect({ label, headers, value, onChange }: { label: string; headers: string[]; value: string; onChange: (value: string) => void }) {
  return (
    <label>
      {label}
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">Not mapped</option>
        {headers.map((header) => (
          <option key={header} value={header}>
            {header}
          </option>
        ))}
      </select>
    </label>
  );
}
