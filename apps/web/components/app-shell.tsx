"use client";

import {
  BarChart3,
  Bell,
  Building2,
  CheckCheck,
  CircleHelp,
  ClipboardCheck,
  GraduationCap,
  Home,
  Menu,
  Network,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  Settings,
  Users,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";

import type { InboxItem } from "@/lib/generated/notify/inbox";
import type { Session } from "@/lib/gateway";

import { BreadcrumbProvider, useBreadcrumb, type Crumb } from "./shell-breadcrumb";

export type ShellArea = "admin" | "employee" | "manager";

type NavItem = {
  href?: string;
  label: string;
  icon: LucideIcon;
};

type NavGroupConfig = {
  title: string;
  items: NavItem[];
};

type AreaConfig = {
  kicker: string;
  groups: NavGroupConfig[];
  searchable: { href: string; label: string }[];
};

const areaConfig: Record<ShellArea, AreaConfig> = {
  admin: {
    kicker: "Admin",
    groups: [
      { title: "Workspace", items: [{ href: "/admin", label: "Home", icon: Home }] },
      {
        title: "Skills management",
        items: [
          { href: "/admin/role-profiles", label: "Role skills profiles", icon: Network },
          { href: "/admin/role-profiles/review", label: "Review & Approvals", icon: CheckCheck },
        ],
      },
      {
        title: "Coming soon",
        items: [
          { label: "Skills Insights", icon: BarChart3 },
          { label: "Learning & Development", icon: GraduationCap },
          { label: "Employees", icon: Users },
          { label: "Organisational Structure", icon: Building2 },
          { label: "Settings", icon: Settings },
        ],
      },
    ],
    searchable: [
      { href: "/admin", label: "Home" },
      { href: "/admin/role-profiles", label: "Role skills profiles" },
      { href: "/admin/role-profiles/review", label: "Review & Approvals" },
    ],
  },
  employee: {
    kicker: "Capability",
    groups: [
      {
        title: "Workspace",
        items: [{ href: "/employee", label: "My capability", icon: ClipboardCheck }],
      },
    ],
    searchable: [{ href: "/employee", label: "My capability" }],
  },
  manager: {
    kicker: "Team",
    groups: [
      {
        title: "Workspace",
        items: [{ href: "/manager", label: "My team", icon: Users }],
      },
    ],
    searchable: [{ href: "/manager", label: "My team" }],
  },
};

export function AppShell({
  area = "admin",
  session,
  notifications,
  notificationError = "",
  children,
}: {
  area?: ShellArea;
  session: Session;
  notifications: InboxItem[];
  notificationError?: string;
  children: ReactNode;
}) {
  return (
    <BreadcrumbProvider>
      <ShellFrame
        area={area}
        session={session}
        notifications={notifications}
        notificationError={notificationError}
      >
        {children}
      </ShellFrame>
    </BreadcrumbProvider>
  );
}

function ShellFrame({
  area,
  session,
  notifications,
  notificationError,
  children,
}: {
  area: ShellArea;
  session: Session;
  notifications: InboxItem[];
  notificationError: string;
  children: ReactNode;
}) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState(false);
  const role = accountLabel(session.roles);
  const crumbs = useTrail(area, pathname);
  const navId = useId();
  const config = areaConfig[area];
  const heading = pageHeading(area, pathname);

  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") {
      return;
    }
    const media = window.matchMedia("(max-width: 1024px)");
    const sync = () => {
      setDrawerMode(media.matches);
      if (!media.matches) {
        setMobileOpen(false);
      }
    };
    sync();
    media.addEventListener("change", sync);
    return () => media.removeEventListener("change", sync);
  }, []);

  useEffect(() => {
    if (!mobileOpen) {
      return;
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setMobileOpen(false);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [mobileOpen]);

  useEffect(() => {
    if (!drawerMode || !mobileOpen) {
      document.body.style.removeProperty("overflow");
      return;
    }
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.removeProperty("overflow");
    };
  }, [drawerMode, mobileOpen]);

  return (
    <div className="app-shell" data-collapsed={collapsed ? "true" : "false"} data-nav={mobileOpen ? "open" : "closed"} data-area={area}>
      <aside
        id={navId}
        className="app-sidebar"
        aria-label="Primary"
        {...(drawerMode && !mobileOpen ? { inert: true, "aria-hidden": true as const } : {})}
      >
        <div className="app-brand">
          <span className="app-brand-compact" aria-hidden="true">
            B
          </span>
          <p className="app-brand-mark">BICMAS</p>
          <p className="app-brand-product">Learn</p>
        </div>
        <nav className="app-nav" aria-label="Sections">
          {config.groups.map((group) => (
            <NavGroup key={group.title} title={group.title} items={group.items} pathname={pathname} />
          ))}
        </nav>
        <div className="app-sidebar-foot">
          <button type="button" className="app-collapse" onClick={() => setCollapsed((value) => !value)} aria-pressed={collapsed}>
            {collapsed ? <PanelLeftOpen size={20} aria-hidden="true" /> : <PanelLeftClose size={20} aria-hidden="true" />}
            <span>{collapsed ? "Expand sidebar" : "Collapse sidebar"}</span>
          </button>
          <ProfileMenu session={session} role={role} />
        </div>
      </aside>
      {mobileOpen ? (
        <button type="button" className="app-backdrop" aria-label="Close navigation" onClick={() => setMobileOpen(false)} />
      ) : null}
      <div className="app-main">
        <header className="app-topbar">
          <button
            type="button"
            className="app-nav-toggle"
            aria-expanded={mobileOpen}
            aria-controls={navId}
            onClick={() => setMobileOpen((value) => !value)}
          >
            <Menu size={20} aria-hidden="true" />
            <span className="app-clip">{mobileOpen ? "Close navigation" : "Open navigation"}</span>
          </button>
          <Breadcrumb crumbs={crumbs} />
          <PageSearch pages={config.searchable} />
          <div className="app-tools">
            <NotificationMenu items={notifications} error={notificationError} />
            <HelpMenu />
            <ProfileMenu session={session} role={role} compact />
          </div>
        </header>
        <div className="app-content">
          {heading ? (
            <header className="app-page-header">
              <p className="kicker">{heading.kicker}</p>
              <h1>{heading.title}</h1>
              {heading.description ? <p className="app-page-description">{heading.description}</p> : null}
            </header>
          ) : null}
          {children}
        </div>
      </div>
    </div>
  );
}

function NavGroup({ title, items, pathname }: { title: string; items: NavItem[]; pathname: string }) {
  return (
    <div className="app-nav-group">
      <p className="app-nav-group-label">{title}</p>
      <ul>
        {items.map((item) => {
          const Icon = item.icon;
          const current = item.href ? currentLink(pathname, item.href) : false;
          const body = (
            <>
              <Icon size={20} aria-hidden="true" />
              <span className="app-nav-label">{item.label}</span>
              {item.href ? null : <span className="app-clip">Not available yet</span>}
            </>
          );
          return (
            <li key={item.label}>
              {item.href ? (
                <Link className="app-nav-link" href={item.href} aria-current={current ? "page" : undefined}>
                  {body}
                </Link>
              ) : (
                <span className="app-nav-link" aria-disabled="true">
                  {body}
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function Breadcrumb({ crumbs }: { crumbs: Crumb[] }) {
  return (
    <nav className="app-breadcrumb" aria-label="Breadcrumb">
      <ol>
        {crumbs.map((crumb, index) => {
          const last = index === crumbs.length - 1;
          return (
            <li key={`${crumb.label}-${index}`} aria-current={last ? "page" : undefined}>
              {crumb.href && !last ? <Link href={crumb.href}>{crumb.label}</Link> : crumb.label}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

function PageSearch({ pages }: { pages: { href: string; label: string }[] }) {
  const router = useRouter();
  const listId = useId();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const matches = pages.filter((item) => item.label.toLowerCase().includes(query.trim().toLowerCase()));
  const visible = open && query.trim().length > 0;

  return (
    <div className="app-search">
      <form
        role="search"
        onSubmit={(event) => {
          event.preventDefault();
          const target = matches[0];
          if (target) {
            router.push(target.href);
            setOpen(false);
            setQuery("");
          }
        }}
      >
        <Search size={20} aria-hidden="true" />
        <input
          type="search"
          value={query}
          placeholder="Search pages"
          aria-label="Search pages"
          aria-controls={visible ? listId : undefined}
          aria-autocomplete="list"
          onChange={(event) => {
            setQuery(event.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
        />
      </form>
      {visible ? (
        <ul id={listId} role="listbox" aria-label="Matching pages">
          {matches.length === 0 ? <li className="app-search-empty">No matching page.</li> : null}
          {matches.map((item) => (
            <li key={item.href}>
              <Link
                href={item.href}
                onClick={() => {
                  setOpen(false);
                  setQuery("");
                }}
              >
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function NotificationMenu({ items, error }: { items: InboxItem[]; error: string }) {
  const label = items.length > 0 ? `Notifications, ${items.length} unread` : "Notifications";
  return (
    <MenuButton
      label={label}
      icon={
        <span className="app-tool-icon">
          <Bell size={20} aria-hidden="true" />
          {items.length > 0 ? <span className="app-count">{items.length}</span> : null}
        </span>
      }
    >
      <p className="app-menu-title">Notifications</p>
      {error ? <p className="app-menu-note">{error}</p> : null}
      {!error && items.length === 0 ? <p className="app-menu-note">No notifications.</p> : null}
      {items.length > 0 ? (
        <ul className="app-menu-list">
          {items.map((item) => (
            <li key={item.id}>
              <strong>{item.title}</strong>
              <span>{item.body}</span>
            </li>
          ))}
        </ul>
      ) : null}
    </MenuButton>
  );
}

function HelpMenu() {
  return (
    <MenuButton label="Help" icon={<CircleHelp size={20} aria-hidden="true" />}>
      <p className="app-menu-title">Help</p>
      <p className="app-menu-note">Help is not available yet.</p>
    </MenuButton>
  );
}

function ProfileMenu({ session, role, compact = false }: { session: Session; role: string; compact?: boolean }) {
  const initials = initialsFor(session.name);
  return (
    <MenuButton
      label={`Account menu for ${session.name}`}
      icon={
        <span className="app-avatar" aria-hidden="true">
          {initials}
        </span>
      }
      className={compact ? "app-profile app-profile-compact" : "app-profile"}
      face={
        compact ? null : (
          <span className="app-profile-text">
            <span className="app-profile-name">{session.name}</span>
            <span className="app-profile-role">{role}</span>
          </span>
        )
      }
    >
      <p className="app-menu-title">{session.name}</p>
      <p className="app-menu-note">{role}</p>
      <form action="/session/sign-out" method="post">
        <button type="submit">Sign out</button>
      </form>
    </MenuButton>
  );
}

function MenuButton({
  label,
  icon,
  className = "",
  face = undefined,
  children,
}: {
  label: string;
  icon: ReactNode;
  className?: string;
  face?: ReactNode;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const pathname = usePathname();

  useEffect(() => {
    setOpen(false);
  }, [pathname]);

  useEffect(() => {
    if (!open) {
      return;
    }
    function onPointer(event: MouseEvent) {
      if (!ref.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onPointer);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointer);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className={`app-menu ${className}`} ref={ref}>
      <button type="button" aria-expanded={open} aria-label={label} onClick={() => setOpen((value) => !value)}>
        {icon}
        {face}
      </button>
      {open ? <div className="app-menu-panel">{children}</div> : null}
    </div>
  );
}

function useTrail(area: ShellArea, pathname: string): Crumb[] {
  const extra = useBreadcrumb();
  if (extra.length > 0) {
    return extra;
  }
  if (area === "employee") {
    return [{ label: "My capability" }];
  }
  if (area === "manager") {
    if (pathname.startsWith("/manager/people/")) {
      return [
        { href: "/manager", label: "My team" },
        { label: "Team member" },
      ];
    }
    return [{ label: "My team" }];
  }
  if (pathname.startsWith("/admin/role-profiles/review")) {
    return [
      { href: "/admin", label: "Home" },
      { label: "Review & Approvals" },
    ];
  }
  if (pathname.startsWith("/admin/role-profiles")) {
    return [
      { href: "/admin", label: "Home" },
      { label: "Role skills profiles" },
    ];
  }
  return [{ label: "Home" }];
}

function currentLink(pathname: string, href: string): boolean {
  if (href === "/admin") {
    return pathname === "/admin";
  }
  if (href === "/admin/role-profiles") {
    return pathname.startsWith("/admin/role-profiles") && !pathname.startsWith("/admin/role-profiles/review");
  }
  if (href === "/employee") {
    return pathname === "/employee" || pathname.startsWith("/employee/");
  }
  if (href === "/manager") {
    return pathname === "/manager" || pathname.startsWith("/manager/");
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

function pageHeading(area: ShellArea, pathname: string): { kicker: string; title: string; description: string } | null {
  if (area === "employee") {
    return {
      kicker: "Capability",
      title: "My capability",
      description: "Complete assigned assessments, review your levels, and record self-assessment evidence.",
    };
  }
  if (area === "manager") {
    if (pathname.startsWith("/manager/people/")) {
      return {
        kicker: "Team",
        title: "Team member",
        description: "Review capability gaps, confirm proposals, and record manager assessments.",
      };
    }
    return {
      kicker: "Team",
      title: "My team",
      description: "See who reports into you and open a person record to review capability.",
    };
  }
  if (pathname.startsWith("/admin/role-profiles/review")) {
    return {
      kicker: "Admin",
      title: "Review & Approvals",
      description: "Manage submitted role skills profiles and configure approval workflows.",
    };
  }
  if (pathname.startsWith("/admin/role-profiles")) {
    return { kicker: "Admin", title: "Role skills profiles", description: "" };
  }
  return { kicker: "Admin", title: "Home", description: "" };
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

function initialsFor(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "U";
  }
  return parts
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}
