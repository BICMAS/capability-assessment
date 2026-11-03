"use client";

import { FileUp, Upload } from "lucide-react";
import { useRouter } from "next/navigation";
import { useId, useRef, useState } from "react";

import { maxOrganogramBytes, organogramUploadIssue } from "./upload-rules";

type ImportResult = {
  file: { id: string; status: string; detail: string; fileName: string };
  accepted: number;
  rejected: { row: number; message: string }[];
  headers: string[];
};

export function OrganogramUpload({ organogramId }: { organogramId: string }) {
  const router = useRouter();
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [progress, setProgress] = useState("");
  const [progressValue, setProgressValue] = useState(0);
  const [error, setError] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragging, setDragging] = useState(false);
  const [titleColumn, setTitleColumn] = useState("");
  const [reportsToColumn, setReportsToColumn] = useState("");
  const [departmentColumn, setDepartmentColumn] = useState("");
  const uploading = progress !== "";

  function choose(next: File | null) {
    setResult(null);
    setError("");
    setProgress("");
    setProgressValue(0);
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
    setProgressValue(0);
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
        const percent = Math.round((event.loaded / event.total) * 100);
        setProgressValue(percent);
        setProgress(`Uploading ${percent}%`);
      }
    };
    request.upload.onload = () => {
      setProgressValue(100);
      setProgress("Processing the file.");
    };
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
  const success = result && result.file.status !== "rejected" && result.accepted > 0;

  return (
    <div className="organo-upload">
      <div className="organo-upload-copy">
        <h3>Upload your organisation structure</h3>
        <p>
          Import an existing organisation chart to automatically structure your departments, positions and reporting
          relationships.
        </p>
        <p className="portal-note">
          Accepted files: PNG, JPG, PDF, XLS, and XLSX up to {formatBytes(maxOrganogramBytes)}. PNG, JPG, and PDF are stored for
          review and are not turned into positions. XLSX can import a title, reporting line, and department. XLS files must be
          saved as XLSX first.
        </p>
      </div>

      <div
        className={dragging ? "organo-drop is-dragging" : "organo-drop"}
        onDragEnter={(event) => {
          event.preventDefault();
          setDragging(true);
        }}
        onDragOver={(event) => event.preventDefault()}
        onDragLeave={(event) => {
          event.preventDefault();
          if (event.currentTarget === event.target) {
            setDragging(false);
          }
        }}
        onDrop={(event) => {
          event.preventDefault();
          setDragging(false);
          choose(event.dataTransfer.files[0] ?? null);
        }}
      >
        <span className="organo-drop-icon" aria-hidden="true">
          <Upload size={28} />
        </span>
        <p className="organo-drop-title">Drag and drop your file here</p>
        <p className="organo-drop-note">or browse from your computer</p>
        <label className="organo-browse" htmlFor={inputId}>
          Browse files
        </label>
        <input
          id={inputId}
          ref={inputRef}
          aria-label="Organogram file"
          type="file"
          accept=".png,.jpg,.jpeg,.pdf,.xls,.xlsx,image/png,image/jpeg,application/pdf"
          onChange={(event) => choose(event.target.files?.[0] ?? null)}
        />
      </div>

      {file ? (
        <div className="organo-file">
          <span className="organo-file-icon" aria-hidden="true">
            <FileUp size={18} />
          </span>
          <div>
            <strong>{file.name}</strong>
            <span>{formatBytes(file.size)}</span>
          </div>
        </div>
      ) : null}

      {uploading ? (
        <div className="organo-progress" role="status" aria-live="polite">
          <div className="organo-progress-bar" aria-hidden="true">
            <span style={{ width: `${progressValue}%` }} />
          </div>
          <p>{progress}</p>
        </div>
      ) : null}

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

      <div className="organo-upload-actions">
        <button type="button" className="primary" disabled={!file || uploading} onClick={() => file && send(file)}>
          Upload file
        </button>
        {error && file ? (
          <button type="button" disabled={uploading} onClick={() => send(file)}>
            Retry upload
          </button>
        ) : null}
      </div>

      {result ? (
        <div className={success ? "organo-result is-success" : "organo-result"} role="status">
          <p>
            <strong>{result.file.fileName}</strong>: {result.file.detail}
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

function formatBytes(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${Math.round(bytes / 1024)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
