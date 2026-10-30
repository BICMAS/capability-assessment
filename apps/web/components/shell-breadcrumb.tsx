"use client";

import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";

export type Crumb = {
  href?: string;
  label: string;
};

type BreadcrumbState = {
  extra: Crumb[];
  setExtra: (items: Crumb[]) => void;
};

const BreadcrumbContext = createContext<BreadcrumbState>({
  extra: [],
  setExtra: () => undefined,
});

export function BreadcrumbProvider({ children }: { children: ReactNode }) {
  const [extra, setExtra] = useState<Crumb[]>([]);
  const value = useMemo(() => ({ extra, setExtra }), [extra]);
  return <BreadcrumbContext.Provider value={value}>{children}</BreadcrumbContext.Provider>;
}

export function useBreadcrumb(): Crumb[] {
  return useContext(BreadcrumbContext).extra;
}

export function SetBreadcrumb({ items }: { items: Crumb[] }) {
  const { setExtra } = useContext(BreadcrumbContext);
  const key = items.map((item) => `${item.href ?? ""}|${item.label}`).join(">");
  useEffect(() => {
    setExtra(items);
    return () => setExtra([]);
  }, [items, key, setExtra]);
  return null;
}
