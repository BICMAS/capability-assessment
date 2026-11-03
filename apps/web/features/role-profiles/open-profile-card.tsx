"use client";

import Link from "next/link";
import { useState } from "react";

import { PendingButton } from "@/components/ui/pending-button";
import { StatusMessage } from "@/components/ui/status-message";

import { openRoleProfileForm } from "./actions";

export function OpenProfileCard({ openError = "" }: { openError?: string }) {
  const [profileId, setProfileId] = useState("");
  const [localError, setLocalError] = useState("");
  const error = localError || openError;

  return (
    <aside className="role-card role-open-profile" aria-labelledby="open-profile-heading">
      <header className="role-card-head">
        <h2 id="open-profile-heading">Open a profile</h2>
        <p>Start a new draft, continue review work, or open an existing role skills profile by id.</p>
      </header>

      <div className="role-open-actions">
        <Link className="ds-button ds-button-primary" href="/admin/role-profiles/new">
          Create a role profile
        </Link>
        <Link className="ds-button ds-button-secondary" href="/admin/role-profiles/review">
          Review inbox
        </Link>
      </div>

      <form
        action={openRoleProfileForm}
        className="framework-form directory-open-form"
        onSubmit={(event) => {
          if (profileId.trim() === "") {
            event.preventDefault();
            setLocalError("Enter a profile id.");
            return;
          }
          setLocalError("");
        }}
      >
        <label htmlFor="open-profile-id">
          Profile id
          <input
            id="open-profile-id"
            name="profileId"
            value={profileId}
            onChange={(event) => {
              setProfileId(event.target.value);
              if (localError) {
                setLocalError("");
              }
            }}
            placeholder="e.g. rp_01H…"
            autoComplete="off"
            spellCheck={false}
            aria-invalid={error ? true : undefined}
            aria-required="true"
            aria-describedby={error ? "open-profile-id-error" : "open-profile-id-help"}
          />
        </label>
        <p id="open-profile-id-help" className="directory-field-help">
          Use the id from a saved draft, review item, or shared profile link.
        </p>
        {error ? (
          <StatusMessage tone="error">
            <span id="open-profile-id-error">{error}</span>
          </StatusMessage>
        ) : null}
        <PendingButton className="primary">Open profile</PendingButton>
      </form>
    </aside>
  );
}
