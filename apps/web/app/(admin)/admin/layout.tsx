import type { ReactNode } from "react";
import { cookies } from "next/headers";

import { AppShell } from "@/components/app-shell";
import { listInbox } from "@/lib/generated/notify/inbox";
import { requireAnyRole } from "@/lib/require-session";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export default async function AdminLayout({ children }: { children: ReactNode }) {
  const session = await requireAnyRole(["admin", "hr", "leader"]);
  const token = (await cookies()).get(SESSION_COOKIE)?.value ?? "";
  const inbox = token
    ? await settle(
        listInbox({ token, baseUrl: process.env.NOTIFY_URL }).then((result) => result.items),
        "Notifications",
      )
    : { ok: false as const, error: "Notifications are unavailable." };

  return (
    <AppShell session={session} notifications={inbox.ok ? inbox.value : []} notificationError={inbox.ok ? "" : inbox.error}>
      {children}
    </AppShell>
  );
}
