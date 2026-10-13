import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

import { fetchSession } from "@/lib/gateway";
import { SESSION_COOKIE, homeFor, roleForPath } from "@/lib/session";

export async function middleware(request: NextRequest) {
  const token = request.cookies.get(SESSION_COOKIE)?.value;
  const session = token ? await fetchSession(token) : null;
  const home = session ? homeFor(session.roles) : null;
  const pathname = request.nextUrl.pathname;
  const required = roleForPath(pathname);

  if (!home) {
    if (required) {
      const redirect = NextResponse.redirect(new URL("/", request.url));
      if (token) {
        redirect.cookies.set(SESSION_COOKIE, "", { ...sessionClear(), maxAge: 0 });
      }
      return redirect;
    }
    return NextResponse.next();
  }

  if (pathname === "/") {
    return NextResponse.redirect(new URL(home, request.url));
  }

  if (required && !session?.roles.includes(required)) {
    return NextResponse.redirect(new URL(home, request.url));
  }

  return NextResponse.next();
}

function sessionClear() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
  };
}

export const config = {
  matcher: [
    "/",
    "/admin",
    "/admin/:path*",
    "/employee",
    "/employee/:path*",
    "/manager",
    "/manager/:path*",
  ],
};
