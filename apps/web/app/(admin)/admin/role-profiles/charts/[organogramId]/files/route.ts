import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { orgDirectoryUrl } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

export async function POST(request: Request, context: { params: Promise<{ organogramId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "Sign in required." }, { status: 401 });
  }
  const { organogramId } = await context.params;
  const incoming = await request.formData();
  const file = incoming.get("file");
  if (!(file instanceof File)) {
    return NextResponse.json({ error: "Choose a file to upload." }, { status: 400 });
  }
  const outbound = new FormData();
  outbound.set("file", file, file.name);
  for (const field of ["titleColumn", "reportsToColumn", "departmentColumn"]) {
    const value = incoming.get(field);
    if (typeof value === "string" && value.length > 0) {
      outbound.set(field, value);
    }
  }
  const response = await fetch(`${orgDirectoryUrl()}/organograms/${encodeURIComponent(organogramId)}/files`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: outbound,
    cache: "no-store",
  });
  const body = await response.text();
  return new NextResponse(body, {
    status: response.status,
    headers: { "Content-Type": response.headers.get("content-type") ?? "text/plain", "Cache-Control": "private, no-store" },
  });
}
