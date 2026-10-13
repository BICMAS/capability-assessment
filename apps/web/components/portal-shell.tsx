import type { ReactNode } from "react";

import type { Session } from "@/lib/gateway";

export function PortalShell({
  title,
  href,
  session,
  rail = false,
  children,
}: {
  title: string;
  href: string;
  session: Session;
  rail?: boolean;
  children: ReactNode;
}) {
  return (
    <div className={rail ? "app app-rail" : "app"}>
      <nav className="nav">
        <p className="mark">BICMAS</p>
        <a href={href} aria-current="page">
          {title}
        </a>
        <div className="nav-session">
          <p className="nav-name">{session.name}</p>
          <form action="/session/sign-out" method="post">
            <button type="submit">Sign out</button>
          </form>
        </div>
      </nav>
      <main className="main">
        <h1>{title}</h1>
        {children}
      </main>
      {rail ? <aside className="rail" /> : null}
    </div>
  );
}
