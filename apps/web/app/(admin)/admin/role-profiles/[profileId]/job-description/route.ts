import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { capabilityFrameworkUrl } from "@/lib/capability-framework";
import { SESSION_COOKIE } from "@/lib/session";

export async function POST(request: Request, context: { params: Promise<{ profileId: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "Sign in required." }, { status: 401 });
  }
  await context.params;
  const incoming = await request.formData();
  const versionId = incoming.get("versionId");
  const file = incoming.get("file");
  if (typeof versionId !== "string" || versionId === "" || !(file instanceof File)) {
    return NextResponse.json({ error: "invalid", detail: "Choose a job description file." }, { status: 400 });
  }
  const outbound = new FormData();
  outbound.set("file", file, file.name);
  const response = await fetch(`${capabilityFrameworkUrl()}/role-profile-versions/${encodeURIComponent(versionId)}/job-description`, {
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
