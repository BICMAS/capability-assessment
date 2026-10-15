import type { ReactNode } from "react";

import { PortalShell } from "@/components/portal-shell";
import { requireSession } from "@/lib/require-session";

export default async function EmployeeLayout({ children }: { children: ReactNode }) {
  const session = await requireSession("employee");
  return (
    <PortalShell title="My capability" kicker="Capability" href="/employee" session={session}>
      {children}
    </PortalShell>
  );
}
