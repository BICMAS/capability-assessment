import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/gateway", () => ({
  requestDevLogin: vi.fn(),
  fetchSession: vi.fn(),
}));

import { POST } from "@/app/session/route";
import { fetchSession, requestDevLogin } from "@/lib/gateway";

const login = vi.mocked(requestDevLogin);
const session = vi.mocked(fetchSession);

function submit(fields: Record<string, string>) {
  const body = new FormData();
  for (const [name, value] of Object.entries(fields)) {
    body.set(name, value);
  }
  return POST(
    new Request("http://localhost/", {
      method: "POST",
      body,
    }),
  );
}

describe("demo sign-in", () => {
  beforeEach(() => {
    login.mockReset();
    session.mockReset();
  });

  it("sends Ada's person id to the gateway and does not send the password", async () => {
    login.mockResolvedValue("token-ada");
    session.mockResolvedValue({
      personId: "ada",
      organisationId: "bicmas",
      roles: ["admin"],
      managerOf: [],
      name: "Ada",
    });

    const response = await submit({ email: "Ada@bicmas.learn", password: "demo" });

    expect(login).toHaveBeenCalledTimes(1);
    expect(login).toHaveBeenCalledWith("ada");
    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("http://localhost/admin");
    expect(response.headers.get("set-cookie")).toContain("session=token-ada");
  });

  it("rejects an unknown email without a session cookie", async () => {
    const response = await submit({ email: "lee@bicmas.learn", password: "demo" });

    expect(login).not.toHaveBeenCalled();
    expect(response.status).toBe(303);
    expect(response.headers.get("location")).toBe("http://localhost/?error=sign-in");
    expect(response.headers.get("set-cookie")).toBeNull();
  });

  it("rejects an empty password without calling the gateway", async () => {
    const response = await submit({ email: "ada@bicmas.learn", password: "   " });

    expect(login).not.toHaveBeenCalled();
    expect(response.headers.get("location")).toBe("http://localhost/?error=sign-in");
    expect(response.headers.get("set-cookie")).toBeNull();
  });

  it("keeps the login page when the gateway does not return a session", async () => {
    login.mockResolvedValue(null);

    const response = await submit({ email: "eli@bicmas.learn", password: "demo" });

    expect(login).toHaveBeenCalledWith("eli");
    expect(response.headers.get("location")).toBe("http://localhost/?error=sign-in");
    expect(response.headers.get("set-cookie")).toBeNull();
  });
});
