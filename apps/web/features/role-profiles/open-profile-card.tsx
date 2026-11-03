import Link from "next/link";

import { PendingButton } from "@/components/ui/pending-button";

import { openRoleProfileForm } from "./actions";

export function OpenProfileCard() {
  return (
    <aside className="role-card role-open-profile" aria-labelledby="open-profile-heading">
      <header className="role-card-head">
        <h2 id="open-profile-heading">Open a profile</h2>
        <p>Create a draft, open a known profile id, or continue in the review inbox.</p>
      </header>
      <div className="role-open-actions">
        <Link className="learn-link directory-create" href="/admin/role-profiles/new">
          Create a role profile
        </Link>
        <Link className="learn-link directory-secondary" href="/admin/role-profiles/review">
          Review inbox
        </Link>
      </div>
      <form action={openRoleProfileForm} className="framework-form">
        <label>
          Profile id
          <input name="profileId" required placeholder="Paste a profile id" />
        </label>
        <PendingButton>Open profile</PendingButton>
      </form>
    </aside>
  );
}
