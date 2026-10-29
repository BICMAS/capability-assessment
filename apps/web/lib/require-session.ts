import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { fetchSession, type Session } from "@/lib/gateway";
import { SESSION_COOKIE, homeFor, type Role } from "@/lib/session";

export async function requireSession(role: Role): Promise<Session> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const session = await fetchSession(token);
  if (!session) {
    redirect("/");
  }
  if (!session.roles.includes(role)) {
    redirect(homeFor(session.roles) ?? "/");
  }
  return session;
}

export async function requireAnyRole(roles: readonly Role[]): Promise<Session> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const session = await fetchSession(token);
  if (!session) {
    redirect("/");
  }
  if (!roles.some((role) => session.roles.includes(role))) {
    redirect(homeFor(session.roles) ?? "/");
  }
  return session;
}
