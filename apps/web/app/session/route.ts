import { NextResponse } from "next/server";

import { fetchSession, requestDevLogin } from "@/lib/gateway";
import { SESSION_COOKIE, homeFor, sessionCookie } from "@/lib/session";

const people = new Set(["ada", "eli", "maya"]);

export async function POST(request: Request) {
  const form = await request.formData();
  const person = form.get("person");
  if (typeof person !== "string" || !people.has(person)) {
    return NextResponse.redirect(new URL("/?error=sign-in", request.url), 303);
  }

  const token = await requestDevLogin(person);
  const session = token ? await fetchSession(token) : null;
  const home = session ? homeFor(session.roles) : null;
  if (!token || !home) {
    return NextResponse.redirect(new URL("/?error=sign-in", request.url), 303);
  }

  const response = NextResponse.redirect(new URL(home, request.url), 303);
  response.cookies.set(SESSION_COOKIE, token, sessionCookie());
  return response;
}
