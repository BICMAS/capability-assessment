import type { ReactNode } from "react";

import type { Session } from "@/lib/gateway";

export function PortalShell({
  title,
  kicker,
  href,
  session,
  children,
}: {
  title: string;
  kicker: string;
  href: string;
  session: Session;
  children: ReactNode;
}) {
  return (
    <div className="app">
      <nav className="nav">
        <div className="nav-brand">
          <p className="mark">BICMAS</p>
          <p className="nav-name">{session.name}</p>
        </div>
        <a href={href} aria-current="page">
          {title}
        </a>
        <form className="nav-session" action="/session/sign-out" method="post">
          <button type="submit">Sign out</button>
        </form>
      </nav>
      <main className="main">
        <p className="kicker">{kicker}</p>
        <h1>{title}</h1>
        {children}
      </main>
    </div>
  );
}
