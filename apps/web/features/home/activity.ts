import type { AuditRecord } from "@/lib/audit";

export function describeActivity(record: AuditRecord, actorName: string): string {
  const name = actorName.trim() || "Someone";
  switch (record.action) {
    case "person.signed_in":
      return `${name} signed in`;
    default:
      return `${name} recorded ${readableAction(record.action)}`;
  }
}

export function activityStatus(record: AuditRecord): string {
  switch (record.action) {
    case "person.signed_in":
      return "Signed in";
    default:
      return readableAction(record.action);
  }
}

function readableAction(action: string): string {
  return action.replaceAll(".", " ").replaceAll("_", " ").trim();
}
