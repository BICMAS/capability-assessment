"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import {
  PlanAlreadyGenerated,
  PlanImmutable,
  VersionNotPublished,
  approvePlan,
  assignPeople,
  editPlanSkill,
  generateAssessmentPlan,
  requestPlanChanges,
  submitPlan,
  type AssignmentResult,
} from "@/lib/generated/assessment-plan/plans";
import { SESSION_COOKIE } from "@/lib/session";

async function sessionToken(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  return token;
}

export async function confirmAssignment(formData: FormData) {
  const frameworkVersionId = text(formData, "frameworkVersionId");
  const roleId = text(formData, "roleId");
  const reason = text(formData, "exceptionReason");
  const personIds = formData.getAll("personId").filter((value): value is string => typeof value === "string" && value.length > 0);
  const mismatched = new Set(
    formData.getAll("mismatchPersonId").filter((value): value is string => typeof value === "string"),
  );
  if (!frameworkVersionId || !roleId || personIds.length === 0) {
    redirect("/admin?assignmentError=Choose%20people%2C%20a%20published%20version%2C%20and%20a%20role.");
  }
  if (mismatched.size > 0 && reason.length === 0) {
    redirect("/admin?assignmentError=A%20title%20mismatch%20needs%20an%20exception%20reason.");
  }

  let results: AssignmentResult[];
  try {
    const batch = await assignPeople(await sessionToken(), {
      frameworkVersionId,
      roleId,
      people: personIds.map((personId) => ({
        personId,
        exceptionReason: mismatched.has(personId) ? reason : undefined,
      })),
    });
    results = batch.results;
  } catch (error) {
    const message = error instanceof VersionNotPublished ? "The framework version is not published." : "The assignment could not be saved.";
    redirect(`/admin?assignmentError=${encodeURIComponent(message)}`);
  }

  const assigned = results.find((result) => result.assignmentId);
  const params = new URLSearchParams();
  if (assigned?.assignmentId) {
    params.set("assignment", assigned.assignmentId);
  }
  params.set("results", Buffer.from(JSON.stringify(results), "utf8").toString("base64url"));
  revalidatePath("/admin");
  redirect(`/admin?${params.toString()}`);
}

export async function generatePlan(formData: FormData) {
  const assignmentId = text(formData, "assignmentId");
  if (!assignmentId) {
    redirect("/admin");
  }
  let planId = "";
  try {
    const plan = await generateAssessmentPlan(await sessionToken(), assignmentId);
    planId = plan.id;
  } catch (error) {
    const message =
      error instanceof PlanAlreadyGenerated
        ? "A generated plan already exists for this assignment."
        : "The plan could not be generated.";
    redirect(`/admin?assignment=${encodeURIComponent(assignmentId)}&planError=${encodeURIComponent(message)}`);
  }
  revalidatePath("/admin");
  redirect(`/admin?assignment=${encodeURIComponent(assignmentId)}&plan=${encodeURIComponent(planId)}`);
}

export async function savePlanSkill(formData: FormData) {
  const assignmentId = text(formData, "assignmentId");
  const planId = text(formData, "planId");
  const skillId = text(formData, "skillId");
  const method = text(formData, "recommendedMethod");
  const instructions = text(formData, "instructions");
  const prompts = indexed(formData, "prompt");
  const levels = indexed(formData, "level");
  const descriptors = indexed(formData, "descriptor");
  const rubric = descriptors.map((descriptor, index) => ({
    level: Number(levels[index] ?? ""),
    descriptor,
  }));
  if (!assignmentId || !planId || !skillId || !method || !instructions || prompts.length === 0 || rubric.some((band) => !band.descriptor || Number.isNaN(band.level))) {
    redirect(planLocation(assignmentId, planId, "The skill could not be saved."));
  }
  let savedId = "";
  try {
    const plan = await editPlanSkill(await sessionToken(), planId, skillId, {
      recommendedMethod: method,
      instructions,
      assessmentItems: prompts.map((prompt) => ({ prompt })),
      rubric,
    });
    savedId = plan.id;
  } catch (error) {
    const message = error instanceof PlanImmutable ? "An approved plan cannot be edited." : "The skill could not be saved.";
    redirect(planLocation(assignmentId, planId, message));
  }
  revalidatePath("/admin");
  redirect(planLocation(assignmentId, savedId));
}

export async function submitPlanForReview(formData: FormData) {
  await movePlan(formData, submitPlan, "The plan could not be submitted.");
}

export async function requestPlanChange(formData: FormData) {
  const assignmentId = text(formData, "assignmentId");
  const planId = text(formData, "planId");
  const reason = text(formData, "reason");
  if (!assignmentId || !planId || !reason) {
    redirect(planLocation(assignmentId, planId, "Request changes with a reason."));
  }
  let savedId = "";
  try {
    const plan = await requestPlanChanges(await sessionToken(), planId, reason);
    savedId = plan.id;
  } catch (error) {
    const message = error instanceof PlanImmutable ? "An approved plan cannot be changed." : "Changes could not be requested.";
    redirect(planLocation(assignmentId, planId, message));
  }
  revalidatePath("/admin");
  redirect(planLocation(assignmentId, savedId));
}

export async function approveAssessmentPlan(formData: FormData) {
  await movePlan(formData, approvePlan, "The plan could not be approved.");
}

async function movePlan(
  formData: FormData,
  action: (token: string, planId: string) => Promise<{ id: string }>,
  failure: string,
) {
  const assignmentId = text(formData, "assignmentId");
  const planId = text(formData, "planId");
  if (!assignmentId || !planId) {
    redirect("/admin");
  }
  let savedId = "";
  try {
    const plan = await action(await sessionToken(), planId);
    savedId = plan.id;
  } catch (error) {
    const message = error instanceof PlanImmutable ? "An approved plan cannot be changed." : failure;
    redirect(planLocation(assignmentId, planId, message));
  }
  revalidatePath("/admin");
  redirect(planLocation(assignmentId, savedId));
}

function planLocation(assignmentId: string, planId: string, error?: string): string {
  const params = new URLSearchParams();
  if (assignmentId) {
    params.set("assignment", assignmentId);
  }
  if (planId) {
    params.set("plan", planId);
  }
  if (error) {
    params.set("planError", error);
  }
  const query = params.toString();
  return query.length > 0 ? `/admin?${query}` : "/admin";
}

function indexed(formData: FormData, name: string): string[] {
  const values: string[] = [];
  for (const [key, value] of formData.entries()) {
    if ((key === name || key.startsWith(`${name}.`)) && typeof value === "string" && value.trim().length > 0) {
      values.push(value.trim());
    }
  }
  return values;
}

function text(formData: FormData, name: string): string {
  const value = formData.get(name);
  return typeof value === "string" ? value.trim() : "";
}
