import { PendingButton } from "@/components/ui/pending-button";
import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { createRoleProfileForm } from "@/features/role-profiles/actions";

export const dynamic = "force-dynamic";

export default async function NewRoleProfilePage({ searchParams }: { searchParams: Promise<{ error?: string }> }) {
  const params = await searchParams;
  return (
    <RoleProfileFrame current="context" error={params.error ?? ""}>
      <form action={createRoleProfileForm} className="framework-form">
        <p>This creates a role identity and the first draft in the capability service. Directory titles are not copied automatically.</p>
        <label>
          Role name
          <input name="name" required />
        </label>
        <PendingButton className="primary">Create draft</PendingButton>
      </form>
    </RoleProfileFrame>
  );
}
