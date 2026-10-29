import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { orgDirectoryUrl } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

export async function GET() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "Sign in required." }, { status: 401 });
  }
  const response = await fetch(`${orgDirectoryUrl()}/organogram-template`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    return new NextResponse((await response.text()) || "The template could not be downloaded.", { status: response.status });
  }
  const body = await response.arrayBuffer();
  return new NextResponse(body, {
    status: 200,
    headers: {
      "Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
      "Content-Disposition": 'attachment; filename="organogram-template.xlsx"',
      "Cache-Control": "private, no-store",
    },
  });
}
