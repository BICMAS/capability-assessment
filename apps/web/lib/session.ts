export const SESSION_COOKIE = "session";

export type Role = "admin" | "employee" | "manager";

const homes: Record<Role, string> = {
  admin: "/admin",
  employee: "/employee",
  manager: "/manager",
};

export function homeFor(roles: readonly string[]): string | null {
  if (roles.includes("admin")) {
    return homes.admin;
  }
  if (roles.includes("employee")) {
    return homes.employee;
  }
  if (roles.includes("manager")) {
    return homes.manager;
  }
  return null;
}

export function roleForPath(pathname: string): Role | null {
  if (pathname === "/admin" || pathname.startsWith("/admin/")) {
    return "admin";
  }
  if (pathname === "/employee" || pathname.startsWith("/employee/")) {
    return "employee";
  }
  if (pathname === "/manager" || pathname.startsWith("/manager/")) {
    return "manager";
  }
  return null;
}

export function sessionCookie() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 12,
  };
}
