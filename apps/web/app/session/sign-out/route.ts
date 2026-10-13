import { NextResponse } from "next/server";

import { SESSION_COOKIE, sessionCookie } from "@/lib/session";

export async function POST(request: Request) {
  const response = NextResponse.redirect(new URL("/", request.url), 303);
  response.cookies.set(SESSION_COOKIE, "", { ...sessionCookie(), maxAge: 0 });
  return response;
}
