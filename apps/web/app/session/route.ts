import { NextResponse } from "next/server";

import { fetchSession, requestDevLogin } from "@/lib/gateway";
import { SESSION_COOKIE, homeFor, sessionCookie } from "@/lib/session";

const demoPeople: Record<string, string> = {
  "ada@bicmas.learn": "ada",
  "eli@bicmas.learn": "eli",
  "maya@bicmas.learn": "maya",
};

export async function POST(request: Request) {
  const form = await request.formData();
  const email = textField(form, "email").toLowerCase();
  const password = textField(form, "password");
  const person = demoPeople[email];
  if (!person || password.length === 0) {
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

function textField(form: FormData, name: string): string {
  const value = form.get(name);
  return typeof value === "string" ? value.trim() : "";
}
