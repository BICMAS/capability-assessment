"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import type { Session } from "@/lib/gateway";

type Area = "admin" | "employee" | "manager";

const links: Record<Area, { href: string; label: string }[]> = {
  admin: [
    { href: "/admin", label: "Home" },
    { href: "/admin/role-profiles", label: "Role skills profiles" },
    { href: "/admin/role-profiles/review", label: "Review" },
  ],
  employee: [{ href: "/employee", label: "My capability" }],
  manager: [{ href: "/manager", label: "My team" }],
};

export function PortalShell({
  area,
  session,
  children,
}: {
  area: Area;
  session: Session;
  children: ReactNode;
}) {
  const pathname = usePathname();
  const heading = pageHeading(area, pathname);

  return (
    <div className="app">
      <nav className="nav">
        <div className="nav-brand">
          <p className="mark">BICMAS</p>
          <p className="nav-product">Learn</p>
        </div>
        <div className="nav-links">
          {links[area].map((link) => (
            <Link key={link.href} href={link.href} aria-current={currentLink(pathname, link.href) ? "page" : undefined}>
              {link.label}
            </Link>
          ))}
        </div>
        <div className="nav-session">
          <p className="nav-name">{session.name}</p>
          <p className="nav-role">{accountLabel(session.roles)}</p>
          <form action="/session/sign-out" method="post">
            <button type="submit">Sign out</button>
          </form>
        </div>
      </nav>
      <main className="main">
        {heading ? (
          <>
            <p className="kicker">{heading.kicker}</p>
            <h1>{heading.title}</h1>
          </>
        ) : null}
        {children}
      </main>
    </div>
  );
}

function currentLink(pathname: string, href: string): boolean {
  if (href === "/admin") {
    return pathname === "/admin";
  }
  if (href === "/admin/role-profiles") {
    return pathname.startsWith("/admin/role-profiles") && !pathname.startsWith("/admin/role-profiles/review");
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

function accountLabel(roles: readonly string[]): string {
  if (roles.includes("admin")) {
    return "Administrator";
  }
  if (roles.includes("hr")) {
    return "HR and L&D";
  }
  if (roles.includes("leader")) {
    return "Business leader";
  }
  if (roles.includes("manager")) {
    return "Line manager";
  }
  if (roles.includes("employee")) {
    return "Employee";
  }
  return "Signed in";
}

function pageHeading(area: Area, pathname: string): { kicker: string; title: string } | null {
  if (area === "employee" || pathname.startsWith("/manager/people/")) {
    return null;
  }
  return pageTitle(area, pathname);
}

function pageTitle(area: Area, pathname: string): { kicker: string; title: string } {
  if (area === "employee") {
    return { kicker: "Capability", title: "My capability" };
  }
  if (area === "manager") {
    return { kicker: "Team", title: "My team" };
  }
  if (pathname.startsWith("/admin/role-profiles/review")) {
    return { kicker: "Admin", title: "Review" };
  }
  if (pathname.startsWith("/admin/role-profiles")) {
    return { kicker: "Admin", title: "Role skills profiles" };
  }
  return { kicker: "Admin", title: "Home" };
}
