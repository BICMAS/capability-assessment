import type { ReactNode } from "react";

import { PortalShell } from "@/components/portal-shell";
import { requireSession } from "@/lib/require-session";

export default async function ManagerLayout({ children }: { children: ReactNode }) {
  const session = await requireSession("manager");
  return (
    <PortalShell area="manager" session={session}>
      {children}
    </PortalShell>
  );
}
