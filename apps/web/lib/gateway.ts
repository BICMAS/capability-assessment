export type Session = {
  personId: string;
  organisationId: string;
  roles: string[];
  managerOf: string[];
  name: string;
};

export function gatewayUrl(): string {
  const configured = process.env.GATEWAY_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8080";
}

export async function requestDevLogin(person: string): Promise<string | null> {
  let response: Response;
  try {
    response = await fetch(`${gatewayUrl()}/dev/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ person }),
      cache: "no-store",
    });
  } catch {
    return null;
  }
  if (!response.ok) {
    return null;
  }
  const payload: unknown = await response.json();
  if (
    typeof payload !== "object" ||
    payload === null ||
    !("token" in payload) ||
    typeof payload.token !== "string" ||
    payload.token.length === 0
  ) {
    return null;
  }
  return payload.token;
}

export async function fetchSession(token: string): Promise<Session | null> {
  let response: Response;
  try {
    response = await fetch(`${gatewayUrl()}/me`, {
      headers: { Authorization: `Bearer ${token}` },
      cache: "no-store",
    });
  } catch {
    return null;
  }
  if (!response.ok) {
    return null;
  }
  const payload: unknown = await response.json();
  if (!isSession(payload)) {
    return null;
  }
  return payload;
}

function isSession(value: unknown): value is Session {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const record = value as Record<string, unknown>;
  if (record.managerOf == null) {
    record.managerOf = [];
  }
  return (
    typeof record.personId === "string" &&
    typeof record.organisationId === "string" &&
    isStringArray(record.roles) &&
    isStringArray(record.managerOf) &&
    typeof record.name === "string"
  );
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === "string");
}
