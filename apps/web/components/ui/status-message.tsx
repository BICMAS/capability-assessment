export function StatusMessage({
  tone = "error",
  children,
}: {
  tone?: "error" | "success" | "info";
  children: React.ReactNode;
}) {
  if (!children) {
    return null;
  }
  const className = tone === "error" ? "status-message is-error" : tone === "success" ? "status-message is-success" : "status-message";
  return (
    <p className={className} role={tone === "error" ? "alert" : "status"}>
      {children}
    </p>
  );
}

export function EmptyState({ children }: { children: React.ReactNode }) {
  return <p className="empty-state">{children}</p>;
}
