// Client methods for contracts/openapi/notify.yaml operation listInbox.

export type InboxItem = {
  id: string;
  organisationId: string;
  personId: string;
  title: string;
  body: string;
  createdAt: string;
};

export type Inbox = {
  items: InboxItem[];
};

export async function listInbox(args: {
  token: string;
  baseUrl?: string;
}): Promise<Inbox> {
  const baseUrl = (args.baseUrl ?? "http://localhost:8082").replace(/\/$/, "");
  const response = await fetch(`${baseUrl}/inbox`, {
    headers: { Authorization: `Bearer ${args.token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`listInbox failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isInbox(payload)) {
    throw new Error("listInbox returned an unexpected payload");
  }
  return payload;
}

function isInbox(value: unknown): value is Inbox {
  if (typeof value !== "object" || value === null || !("items" in value)) {
    return false;
  }
  const items = value.items;
  return Array.isArray(items) && items.every(isInboxItem);
}

function isInboxItem(value: unknown): value is InboxItem {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  return (
    typeof item.id === "string" &&
    typeof item.organisationId === "string" &&
    typeof item.personId === "string" &&
    typeof item.title === "string" &&
    typeof item.body === "string" &&
    typeof item.createdAt === "string"
  );
}
