import type { RoleProfile, RoleProfileVersion } from "@/lib/role-profile";

export const workflowStages = [
  {
    id: "organogram",
    number: 1,
    title: "Upload Organogram",
    description: "Upload a spreadsheet, build the chart, or import the template.",
  },
  {
    id: "role",
    number: 2,
    title: "Provide Role Information",
    description: "Select the organogram position and record the purpose, outcomes, responsibilities, and context for that role.",
  },
  {
    id: "clarify",
    number: 3,
    title: "AI Clarification Questions",
    description: "Answer follow-up questions when the recorded role information is missing or ambiguous.",
  },
  {
    id: "generate",
    number: 4,
    title: "Generate Skills Profile",
    description: "Generate a draft skills profile from the recorded role context.",
  },
  {
    id: "skills",
    number: 5,
    title: "Review and Customise Skills",
    description: "Review, edit, add, or remove skills before the draft is submitted.",
  },
  {
    id: "approve",
    number: 6,
    title: "Approve and Save",
    description: "HR/L&D, the line manager, and a business leader approve in the configured order. The approved version stays immutable.",
  },
  {
    id: "insights",
    number: 7,
    title: "Use the Insights",
    description: "Read assessment gaps for a published version. The result is not an employment decision.",
  },
] as const;

export type StageId = (typeof workflowStages)[number]["id"];
export type StageStatus = "completed" | "active" | "pending" | "blocked";

export type WorkflowEvidence = {
  organogramCount: number | null;
  clarificationComplete: boolean | null;
};

export type StageView = {
  id: StageId;
  number: number;
  title: string;
  description: string;
  status: StageStatus;
  done: boolean;
  current: boolean;
  href: string;
  tone: "neutral" | "ai" | "attention";
};

const approved = new Set(["profile_published", "profile_superseded"]);
const customised = new Set(["profile_in_review", "profile_published", "profile_superseded"]);

export function buildWorkflow(input: {
  profile?: RoleProfile | null;
  current: StageId;
  evidence?: WorkflowEvidence | null;
}): StageView[] {
  const profile = input.profile ?? null;
  const version = profile?.version ?? null;
  const evidence = input.evidence ?? { organogramCount: null, clarificationComplete: null };
  const roleRecorded = version ? roleInformationRecorded(version) : false;
  const generated = version ? profileGenerated(version) : false;
  const skillsReady = version ? version.skills.length > 0 && customised.has(version.status) : false;
  const approvalRecorded = version ? approved.has(version.status) : false;
  const published = version?.status === "profile_published";

  return workflowStages.map((stage) => {
    const progress = progressFor(stage.id, {
      profile,
      version,
      current: input.current,
      organogramCount: evidence.organogramCount,
      clarificationComplete: evidence.clarificationComplete,
      roleRecorded,
      generated,
      skillsReady,
      approvalRecorded,
      published,
    });
    const tone = stage.id === "clarify" || stage.id === "generate" ? "ai" : stage.id === "approve" && version?.status === "profile_in_review" ? "attention" : "neutral";
    return {
      ...stage,
      status: progress.status,
      done: progress.done,
      current: input.current === stage.id,
      href: progress.href,
      tone,
    };
  });
}

function progressFor(
  id: StageId,
  input: {
    profile: RoleProfile | null;
    version: RoleProfileVersion | null;
    current: StageId;
    organogramCount: number | null;
    clarificationComplete: boolean | null;
    roleRecorded: boolean;
    generated: boolean;
    skillsReady: boolean;
    approvalRecorded: boolean;
    published: boolean;
  },
): { status: StageStatus; done: boolean; href: string } {
  const profileId = input.profile?.id ?? "";
  const current = input.current === id;
  if (id === "organogram") {
    const done = input.organogramCount !== null && input.organogramCount > 0;
    return { done, href: "/admin/role-profiles", status: statusFor(done, false, current) };
  }
  if (id === "role") {
    const href = profileId ? `/admin/role-profiles/${profileId}/context` : current ? "/admin/role-profiles/new" : "";
    return { done: input.roleRecorded, href, status: statusFor(input.roleRecorded, href === "", current) };
  }
  if (id === "clarify") {
    const href = profileId ? `/admin/role-profiles/${profileId}/clarify` : "";
    const done = input.clarificationComplete === true;
    return { done, href, status: statusFor(done, href === "", current) };
  }
  if (id === "generate") {
    const href = profileId ? `/admin/role-profiles/${profileId}/generation` : "";
    return { done: input.generated, href, status: statusFor(input.generated, href === "", current) };
  }
  if (id === "skills") {
    const href = profileId ? `/admin/role-profiles/${profileId}` : "";
    return { done: input.skillsReady, href, status: statusFor(input.skillsReady, href === "", current) };
  }
  if (id === "approve") {
    const href = profileId ? `/admin/role-profiles/${profileId}/review` : current ? "/admin/role-profiles/review" : "";
    return { done: input.approvalRecorded, href, status: statusFor(input.approvalRecorded, href === "", current) };
  }
  const href = input.published && profileId ? `/admin/role-profiles/${profileId}/insights` : "";
  return { done: false, href, status: statusFor(false, href === "", current) };
}

function statusFor(done: boolean, blocked: boolean, current: boolean): StageStatus {
  if (blocked) {
    return "blocked";
  }
  if (current) {
    return "active";
  }
  if (done) {
    return "completed";
  }
  return "pending";
}

export function roleInformationRecorded(version: RoleProfileVersion): boolean {
  return (
    version.purpose.trim() !== "" &&
    version.responsibilities.trim() !== "" &&
    version.businessOutcomes.trim() !== "" &&
    version.decisionScope.trim() !== "" &&
    version.reportingLevel.trim() !== "" &&
    version.teamSize !== null &&
    version.teamSize > 0 &&
    (version.positionId ?? "").trim() !== ""
  );
}

function profileGenerated(version: RoleProfileVersion): boolean {
  return Boolean(version.ai.generatedAt) || version.skills.some((skill) => skill.provenance === "ai_suggested");
}

export function stageStateLabel(stage: Pick<StageView, "status" | "done">): string {
  if (stage.status === "active" && stage.done) {
    return "Current, completed";
  }
  if (stage.status === "active") {
    return "Current";
  }
  if (stage.status === "completed") {
    return "Completed";
  }
  if (stage.status === "blocked") {
    return "Not available yet";
  }
  return "Not started";
}
