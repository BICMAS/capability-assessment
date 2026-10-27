import Link from "next/link";

const steps = [
  { id: "directory", label: "Directory", href: "/admin/role-profiles" },
  { id: "context", label: "Role context", href: "context" },
  { id: "sources", label: "Sources", href: "sources" },
  { id: "clarify", label: "Clarification", href: "clarify" },
  { id: "generation", label: "Generation", href: "generation" },
  { id: "skills", label: "Skills", href: "" },
  { id: "review", label: "Review", href: "review" },
  { id: "history", label: "History", href: "history" },
] as const;

export type WorkflowStep = (typeof steps)[number]["id"];

export function WorkflowStepper({ profileId, current }: { profileId?: string; current: WorkflowStep }) {
  return (
    <nav className="workflow-nav" aria-label="Role profile workflow">
      <ol>
        {steps.map((step, index) => {
          const href = step.id === "directory" ? step.href : profileId ? `/admin/role-profiles/${profileId}${step.href ? `/${step.href}` : ""}` : "";
          return (
            <li key={step.id}>
              {href ? (
                <Link href={href} aria-current={step.id === current ? "step" : undefined}>
                  {index + 1}. {step.label}
                </Link>
              ) : (
                <span aria-current={step.id === current ? "step" : undefined}>
                  {index + 1}. {step.label}
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
