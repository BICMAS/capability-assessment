"use client";

import Link from "next/link";
import { useRef, useState } from "react";

import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile, RoleProfileVersion } from "@/lib/role-profile";

import { saveRoleContext } from "./actions";
import { roleInformationRecorded } from "./workflow";

const levels = [
  ["", "Not set"],
  ["individual", "Individual"],
  ["team_lead", "Team lead"],
  ["manager", "Manager"],
  ["director", "Director"],
  ["executive", "Executive"],
] as const;

export type PositionChoice = {
  organogramId: string;
  organogramName: string;
  positionId: string;
  title: string;
  departmentName: string;
  reportsToTitle: string;
};

export type FrameworkChoice = { id: string; label: string };

export function ContextForm({
  profile,
  positions,
  frameworks,
  directoryError,
  requestedOrganogramId = "",
  requestedPositionId = "",
}: {
  profile: RoleProfile;
  positions: PositionChoice[];
  frameworks: FrameworkChoice[];
  directoryError: string;
  requestedOrganogramId?: string;
  requestedPositionId?: string;
}) {
  const version = profile.version;
  const editable = version.status === "profile_draft";
  const dirty = useRef(false);
  const [notice, setNotice] = useState("");
  const [jobDescription, setJobDescription] = useState(jobDescriptionText(version));
  const [jobFileId, setJobFileId] = useState("");
  const [jobFileName, setJobFileName] = useState("");
  const [uploadError, setUploadError] = useState("");
  const [uploadProgress, setUploadProgress] = useState("");
  const organogramId = version.organogramId || requestedOrganogramId;
  const positionId = version.positionId || requestedPositionId;
  const selected = organogramId && positionId ? `${organogramId}:${positionId}` : "";
  const choices = withSavedPosition(positions, organogramId, positionId);
  const selectedPosition = choices.find((position) => position.organogramId === organogramId && position.positionId === positionId);
  const missing = missingRoleInformation(version);
  const recorded = roleInformationRecorded(version);

  function markDirty() {
    dirty.current = true;
  }

  function upload(file: File) {
    if (file.size > 5 * 1024 * 1024) {
      setUploadError("The file must be 5 MB or smaller.");
      return;
    }
    if (!/\.(txt|pdf|doc|docx)$/i.test(file.name)) {
      setUploadError("Upload a TXT, PDF, DOC, or DOCX file.");
      return;
    }
    setUploadError("");
    setUploadProgress("Uploading 0%");
    const body = new FormData();
    body.set("file", file);
    body.set("versionId", version.id);
    const request = new XMLHttpRequest();
    request.open("POST", `/admin/role-profiles/${encodeURIComponent(profile.id)}/job-description`);
    request.upload.onprogress = (event) => {
      if (event.lengthComputable) {
        setUploadProgress(`Uploading ${Math.round((event.loaded / event.total) * 100)}%`);
      }
    };
    request.onload = () => {
      setUploadProgress("");
      let payload: { id?: string; fileName?: string; detail?: string; text?: string; error?: string } = {};
      try {
        payload = JSON.parse(request.responseText) as typeof payload;
      } catch {
        setUploadError("The upload response could not be read.");
        return;
      }
      if (request.status !== 201 || !payload.id) {
        setUploadError(payload.detail || "The job description could not be stored.");
        return;
      }
      setJobFileId(payload.id);
      setJobFileName(payload.fileName || file.name);
      if (payload.text && jobDescription.trim() === "") {
        setJobDescription(payload.text);
      }
      dirty.current = true;
      setNotice(payload.detail || "The file is stored. Save the role information to keep it with this position.");
    };
    request.onerror = () => {
      setUploadProgress("");
      setUploadError("The upload did not reach the server. Try again.");
    };
    request.send(body);
  }

  return (
    <form
      action={saveRoleContext}
      className="framework-form"
      onChange={markDirty}
      onSubmit={() => {
        dirty.current = false;
      }}
    >
      <input type="hidden" name="profileId" value={profile.id} />
      <input type="hidden" name="versionId" value={version.id} />
      <input type="hidden" name="expectedRevision" value={version.revision} />
      <input type="hidden" name="positionSelection" value="present" />
      <input type="hidden" name="savedOrganogramId" value={version.organogramId ?? ""} />
      <input type="hidden" name="savedPositionId" value={version.positionId ?? ""} />
      <input type="hidden" name="jobFileId" value={jobFileId} />
      <input type="hidden" name="jobFileName" value={jobFileName} />
      {preservedSources(version).map((source) => (
        <span key={source.id}>
          <input type="hidden" name="sourceTitle" value={source.title} />
          <input type="hidden" name="sourceUri" value={source.referenceUri} />
          <input type="hidden" name="sourceKind" value={source.kind === "job_description" ? "job_description" : "reference"} />
          <input type="hidden" name="sourceBody" value={source.body ?? ""} />
          <input type="hidden" name="sourceOrigin" value={source.origin === "imported" || source.origin === "ai_suggested" ? source.origin : "user"} />
        </span>
      ))}
      <p>
        Role title <strong>{profile.roleName}</strong>. Status <strong>{version.status}</strong>. Revision {version.revision}.
      </p>
      {directoryError ? <p className="role-missing">{directoryError}</p> : null}
      {missing.length > 0 ? (
        <div className="role-missing">
          <p>Save these before clarification. You can save a partial draft and return later.</p>
          <ul>
            {missing.map((field) => (
              <li key={field}>{field}</li>
            ))}
          </ul>
        </div>
      ) : (
        <p>The required role information is saved for this position.</p>
      )}
      <label>
        Organogram position
        <select name="positionChoice" defaultValue={selected} disabled={!editable}>
          <option value="">Select a position</option>
          {choices.map((position) => (
            <option key={`${position.organogramId}:${position.positionId}`} value={`${position.organogramId}:${position.positionId}`}>
              {position.organogramName}: {position.title}
            </option>
          ))}
        </select>
      </label>
      {selectedPosition ? (
        <p>
          Department {selectedPosition.departmentName || "not recorded"}. Reporting line {selectedPosition.reportsToTitle || "not recorded"}. These come from the organogram and are not copied into a second department record.
        </p>
      ) : (
        <p>Select a position to show its department and reporting line.</p>
      )}
      <label>
        Business unit
        <input name="businessUnit" defaultValue={version.businessUnit ?? ""} disabled={!editable} />
      </label>
      <label>
        Role purpose
        <textarea name="purpose" defaultValue={version.purpose} disabled={!editable} rows={3} />
      </label>
      <label>
        Job description text
        <textarea name="jobDescription" value={jobDescription} onChange={(event) => setJobDescription(event.target.value)} disabled={!editable} rows={5} />
      </label>
      {editable ? (
        <label>
          Job description file
          <input
            type="file"
            accept=".txt,.pdf,.doc,.docx,text/plain,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                upload(file);
              }
            }}
          />
        </label>
      ) : null}
      {uploadProgress ? <p>{uploadProgress}</p> : null}
      {uploadError ? <p className="role-missing">{uploadError}</p> : null}
      {jobFileName ? <p>Stored file: {jobFileName}. Save the role information to attach it. PDF and Word files are stored without extracting text.</p> : null}
      {storedFiles(version).map((source) => (
        <p key={source.id}>
          <a href={`/admin/role-profiles/${profile.id}/job-description/${source.referenceUri.slice("file:".length)}?version=${encodeURIComponent(version.id)}`}>{source.title}</a>
        </p>
      ))}
      <label>
        Supervisory responsibilities
        <textarea name="responsibilities" defaultValue={version.responsibilities} disabled={!editable} rows={3} />
      </label>
      <label>
        Key business objectives
        <textarea name="businessOutcomes" defaultValue={version.businessOutcomes} disabled={!editable} rows={3} />
      </label>
      <label>
        Decisions made independently
        <textarea name="decisionScope" defaultValue={version.decisionScope} disabled={!editable} rows={3} />
      </label>
      <label>
        Level of responsibility
        <select name="reportingLevel" defaultValue={version.reportingLevel} disabled={!editable}>
          {levels.map(([value, label]) => (
            <option key={value || "empty"} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <label>
        Industry
        <input name="industry" defaultValue={version.industry} disabled={!editable} />
      </label>
      <label>
        Sub-industry
        <input name="subIndustry" defaultValue={version.subIndustry ?? ""} disabled={!editable} />
      </label>
      <label>
        Geographic or operating scope
        <input name="region" defaultValue={version.region} disabled={!editable} />
      </label>
      <label>
        Team size
        <input name="teamSize" inputMode="numeric" defaultValue={version.teamSize ?? ""} disabled={!editable} />
      </label>
      <label>
        Existing competency framework
        <select name="frameworkVersionId" defaultValue={version.frameworkVersionId} disabled={!editable}>
          <option value="">None recorded</option>
          {frameworks.map((framework) => (
            <option key={framework.id} value={framework.id}>
              {framework.label}
            </option>
          ))}
        </select>
      </label>
      <label>
        Additional organisational context
        <textarea name="organisationalContext" defaultValue={version.organisationalContext ?? ""} disabled={!editable} rows={3} />
      </label>
      {notice ? <p>{notice}</p> : null}
      {editable ? <PendingButton className="primary">Save role information</PendingButton> : <p>Published and in-review versions are read here. Start the next draft from history to edit.</p>}
      {recorded ? (
        <Link
          href={`/admin/role-profiles/${profile.id}/clarify`}
          onClick={(event) => {
            if (dirty.current) {
              event.preventDefault();
              setNotice("Save the role information before opening clarification.");
            }
          }}
        >
          Continue to clarification
        </Link>
      ) : (
        <p>Clarification stays closed until the required fields are saved.</p>
      )}
    </form>
  );
}

function jobDescriptionText(version: RoleProfileVersion): string {
  return version.sources.find((source) => source.referenceUri === "entered:job-description")?.body ?? "";
}

function preservedSources(version: RoleProfileVersion) {
  return version.sources.filter((source) => source.referenceUri !== "entered:job-description");
}

function storedFiles(version: RoleProfileVersion) {
  return version.sources.filter((source) => source.referenceUri.startsWith("file:"));
}

function withSavedPosition(positions: PositionChoice[], organogramId: string, positionId: string): PositionChoice[] {
  if (!organogramId || !positionId || positions.some((position) => position.organogramId === organogramId && position.positionId === positionId)) {
    return positions;
  }
  return [
    ...positions,
    { organogramId, organogramName: "Saved organogram", positionId, title: "Saved position", departmentName: "", reportsToTitle: "" },
  ];
}

function missingRoleInformation(version: RoleProfileVersion): string[] {
  const missing: string[] = [];
  if (!(version.positionId ?? "").trim()) {
    missing.push("Organogram position");
  }
  if (!version.purpose.trim()) {
    missing.push("Role purpose");
  }
  if (!version.responsibilities.trim()) {
    missing.push("Supervisory responsibilities");
  }
  if (!version.businessOutcomes.trim()) {
    missing.push("Key business objectives");
  }
  if (!version.decisionScope.trim()) {
    missing.push("Decisions made independently");
  }
  if (!version.reportingLevel.trim()) {
    missing.push("Level of responsibility");
  }
  if (version.teamSize === null || version.teamSize <= 0) {
    missing.push("Team size");
  }
  return missing;
}
