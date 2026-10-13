import type { ReactNode } from "react";

import { PortalShell } from "@/components/portal-shell";
import { requireSession } from "@/lib/require-session";

export default async function AdminLayout({ children }: { children: ReactNode }) {
  const session = await requireSession("admin");
  return (
    <PortalShell title="Frameworks" href="/admin" session={session}>
      {children}
    </PortalShell>
  );
}
