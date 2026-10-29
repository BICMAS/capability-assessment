import { PendingButton } from "@/components/ui/pending-button";
import { createRoleProfileForm } from "@/features/role-profiles/actions";
import { WorkflowShell } from "@/features/role-profiles/frame";
import { sessionToken } from "@/lib/session-token";

export const dynamic = "force-dynamic";

export default async function NewRoleProfilePage({ searchParams }: { searchParams: Promise<{ error?: string; title?: string; position?: string; organogram?: string }> }) {
  const params = await searchParams;
  const token = await sessionToken();
  const title = (params.title ?? "").trim().slice(0, 200);
  const position = (params.position ?? "").trim().slice(0, 80);
  const organogram = (params.organogram ?? "").trim().slice(0, 80);
  return (
    <WorkflowShell token={token} current="role" error={params.error ?? ""}>
      <form action={createRoleProfileForm} className="framework-form">
        <p>This creates a role identity and the first draft in the capability service. The organogram position is linked on the next step. Department and reporting line stay in the directory.</p>
        {title ? <p>Selected organogram position: {title}</p> : null}
        <input type="hidden" name="positionId" value={position} />
        <input type="hidden" name="organogramId" value={organogram} />
        <label>
          Role title
          <input name="name" required defaultValue={title} />
        </label>
        <PendingButton className="primary">Create draft</PendingButton>
      </form>
    </WorkflowShell>
  );
}
