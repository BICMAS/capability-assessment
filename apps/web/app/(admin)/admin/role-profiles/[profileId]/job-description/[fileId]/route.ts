import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { capabilityFrameworkUrl } from "@/lib/capability-framework";
import { SESSION_COOKIE } from "@/lib/session";

export async function GET(request: Request, context: { params: Promise<{ profileId: string; fileId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "Sign in required." }, { status: 401 });
  }
  const { fileId } = await context.params;
  const versionId = new URL(request.url).searchParams.get("version") ?? "";
  if (versionId === "") {
    return NextResponse.json({ error: "invalid", detail: "The role version is missing." }, { status: 400 });
  }
  const response = await fetch(
    `${capabilityFrameworkUrl()}/role-profile-versions/${encodeURIComponent(versionId)}/job-description/${encodeURIComponent(fileId)}`,
    { headers: { Authorization: `Bearer ${token}` }, cache: "no-store" },
  );
  const body = await response.arrayBuffer();
  const headers = new Headers();
  headers.set("Content-Type", response.headers.get("content-type") ?? "application/octet-stream");
  headers.set("Cache-Control", "private, no-store");
  headers.set("X-Content-Type-Options", "nosniff");
  const disposition = response.headers.get("content-disposition");
  if (disposition) {
    headers.set("Content-Disposition", disposition);
  }
  return new NextResponse(body, { status: response.status, headers });
}
