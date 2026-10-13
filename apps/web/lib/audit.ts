export type AuditRecord = {
  actorPersonId: string;
  organisationId: string;
  action: string;
  entityType: string;
  entityId: string;
  occurredAt: string;
};

export function auditUrl(): string {
  const configured = process.env.AUDIT_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8081";
}

export async function listAuditRecords(token: string): Promise<AuditRecord[]> {
  const response = await fetch(`${auditUrl()}/records`, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`/records failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!Array.isArray(payload) || !payload.every(isRecord)) {
    throw new Error("/records returned an unexpected payload");
  }
  return payload;
}

function isRecord(value: unknown): value is AuditRecord {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const record = value as Record<string, unknown>;
  return (
    typeof record.actorPersonId === "string" &&
    typeof record.organisationId === "string" &&
    typeof record.action === "string" &&
    typeof record.entityType === "string" &&
    typeof record.entityId === "string" &&
    typeof record.occurredAt === "string"
  );
}
