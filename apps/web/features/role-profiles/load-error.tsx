import { StatusMessage } from "@/components/ui/status-message";
import { roleProfileFailure } from "@/lib/role-profile";

export function ProfileLoadError({ error }: { error: unknown }) {
  const mapped = roleProfileFailure(error);
  return <StatusMessage tone="error">{mapped?.message ?? "The profile could not be loaded."}</StatusMessage>;
}
