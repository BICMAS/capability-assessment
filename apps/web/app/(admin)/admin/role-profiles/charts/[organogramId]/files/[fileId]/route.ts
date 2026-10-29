import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { orgDirectoryUrl } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

export async function GET(_request: Request, context: { params: Promise<{ organogramId: string; fileId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "Sign in required." }, { status: 401 });
  }
  const { organogramId, fileId } = await context.params;
  const response = await fetch(`${orgDirectoryUrl()}/organograms/${encodeURIComponent(organogramId)}/files/${encodeURIComponent(fileId)}`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    return new NextResponse((await response.text()) || "The file could not be downloaded.", { status: response.status });
  }
  const body = await response.arrayBuffer();
  return new NextResponse(body, {
    status: 200,
    headers: {
      "Content-Type": response.headers.get("content-type") ?? "application/octet-stream",
      "Content-Disposition": response.headers.get("content-disposition") ?? "attachment",
      "X-Content-Type-Options": "nosniff",
      "Cache-Control": "private, no-store",
    },
  });
}
