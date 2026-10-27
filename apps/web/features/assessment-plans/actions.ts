"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { fail, ok, type ActionResult } from "@/lib/action-result";
import {
  PlanAlreadyGenerated,
  PlanImmutable,
  approvePlan,
  confirmAssessmentSession,
  editPlanSkill,
  generateAssessmentPlan,
  readPlan,
  requestPlanChanges,
  submitPlan,
} from "@/lib/generated/assessment-plan/plans";
import { USER_MESSAGES } from "@/lib/errors";
import { indexed, parseForm, text } from "@/lib/form-data";
import { asFormAction } from "@/lib/form-action";
import { runAction } from "@/lib/run-action";
import { sessionToken } from "@/lib/session-token";

const planIdsSchema = z.object({
  assignmentId: z.string().min(1),
  planId: z.string().min(1),
});

const generateSchema = z.object({
  assignmentId: z.string().min(1, "Choose an assignment."),
});

const skillSchema = z.object({
  assignmentId: z.string().min(1),
  planId: z.string().min(1),
  skillId: z.string().min(1),
  recommendedMethod: z.string().min(1).max(80),
  instructions: z.string().min(1).max(8000),
  prompts: z.array(z.string().min(1).max(2000)).min(1),
  levels: z.array(z.string().min(1)),
  descriptors: z.array(z.string().min(1).max(2000)),
  expectedVersionNumber: z.coerce.number().int().positive().optional(),
});

const changeSchema = z.object({
  assignmentId: z.string().min(1),
  planId: z.string().min(1),
  reason: z.string().min(1, "Enter a reason.").max(280),
});

const confirmSchema = z.object({
  assignmentId: z.string().optional(),
  planId: z.string().optional(),
  sessionId: z.string().min(1, "That session was not found."),
});

export async function generatePlan(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(generateSchema, { assignmentId: text(formData, "assignmentId") });
    if ("error" in parsed) {
      redirect("/admin");
    }
    const { assignmentId } = parsed.data;
    let planId = "";
    try {
      const plan = await generateAssessmentPlan(await sessionToken(), assignmentId);
      planId = plan.id;
    } catch (error) {
      const message =
        error instanceof PlanAlreadyGenerated
          ? "A generated plan already exists for this assignment."
          : USER_MESSAGES.unavailable;
      redirect(`/admin?assignment=${encodeURIComponent(assignmentId)}&planError=${encodeURIComponent(message)}`);
    }
    revalidatePath("/admin");
    redirect(`/admin?assignment=${encodeURIComponent(assignmentId)}&plan=${encodeURIComponent(planId)}`);
  });
}

export async function savePlanSkill(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(skillSchema, {
      assignmentId: text(formData, "assignmentId"),
      planId: text(formData, "planId"),
      skillId: text(formData, "skillId"),
      recommendedMethod: text(formData, "recommendedMethod"),
      instructions: text(formData, "instructions"),
      prompts: indexed(formData, "prompt"),
      levels: indexed(formData, "level"),
      descriptors: indexed(formData, "descriptor"),
      expectedVersionNumber: formData.get("expectedVersionNumber") || undefined,
    });
    if ("error" in parsed) {
      const assignmentId = text(formData, "assignmentId");
      const planId = text(formData, "planId");
      redirect(planLocation(assignmentId, planId, USER_MESSAGES.validation));
    }
    const { assignmentId, planId, skillId, recommendedMethod, instructions, prompts, levels, descriptors, expectedVersionNumber } =
      parsed.data;
    const rubric = descriptors.map((descriptor, index) => ({
      level: Number(levels[index] ?? ""),
      descriptor,
    }));
    if (rubric.some((band) => !band.descriptor || Number.isNaN(band.level))) {
      redirect(planLocation(assignmentId, planId, USER_MESSAGES.validation));
    }

    const token = await sessionToken();
    if (expectedVersionNumber != null) {
      const current = await readPlan({ token, planId });
      if (!current) {
        redirect(planLocation(assignmentId, planId, USER_MESSAGES.not_found));
      }
      if (current.versionNumber !== expectedVersionNumber) {
        redirect(planLocation(assignmentId, planId, USER_MESSAGES.conflict));
      }
    }

    let savedId = "";
    try {
      const plan = await editPlanSkill(token, planId, skillId, {
        recommendedMethod,
        instructions,
        assessmentItems: prompts.map((prompt) => ({ prompt })),
        rubric,
      });
      savedId = plan.id;
    } catch (error) {
      const message = error instanceof PlanImmutable ? "An approved plan cannot be edited." : USER_MESSAGES.unavailable;
      redirect(planLocation(assignmentId, planId, message));
    }
    revalidatePath("/admin");
    redirect(planLocation(assignmentId, savedId));
  });
}

export async function submitPlanForReview(formData: FormData): Promise<void> {
  await movePlan(formData, submitPlan, "The plan could not be submitted.");
}

export async function requestPlanChange(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(changeSchema, {
      assignmentId: text(formData, "assignmentId"),
      planId: text(formData, "planId"),
      reason: text(formData, "reason"),
    });
    if ("error" in parsed) {
      redirect(
        planLocation(text(formData, "assignmentId"), text(formData, "planId"), "Request changes with a reason."),
      );
    }
    const { assignmentId, planId, reason } = parsed.data;
    let savedId = "";
    try {
      const plan = await requestPlanChanges(await sessionToken(), planId, reason);
      savedId = plan.id;
    } catch (error) {
      const message = error instanceof PlanImmutable ? "An approved plan cannot be changed." : USER_MESSAGES.unavailable;
      redirect(planLocation(assignmentId, planId, message));
    }
    revalidatePath("/admin");
    redirect(planLocation(assignmentId, savedId));
  });
}

export async function approveAssessmentPlan(formData: FormData): Promise<void> {
  await movePlan(formData, approvePlan, "The plan could not be approved.");
}

export async function confirmProposal(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(confirmSchema, {
      assignmentId: text(formData, "assignmentId") || undefined,
      planId: text(formData, "planId") || undefined,
      sessionId: text(formData, "sessionId"),
    });
    if ("error" in parsed) {
      redirect(planLocation(text(formData, "assignmentId"), text(formData, "planId"), "That session was not found."));
    }
    const { assignmentId, planId, sessionId } = parsed.data;
    try {
      await confirmAssessmentSession(await sessionToken(), sessionId);
    } catch {
      redirect(planLocation(assignmentId ?? "", planId ?? "", "The proposal could not be confirmed."));
    }
    revalidatePath("/admin");
    revalidatePath("/employee");
    redirect(planLocation(assignmentId ?? "", planId ?? ""));
  });
}

/** Typed result variant for unit tests and useActionState callers. */
export async function savePlanSkillResult(formData: FormData): Promise<ActionResult<{ planId: string }>> {
  const parsed = parseForm(skillSchema, {
    assignmentId: text(formData, "assignmentId"),
    planId: text(formData, "planId"),
    skillId: text(formData, "skillId"),
    recommendedMethod: text(formData, "recommendedMethod"),
    instructions: text(formData, "instructions"),
    prompts: indexed(formData, "prompt"),
    levels: indexed(formData, "level"),
    descriptors: indexed(formData, "descriptor"),
    expectedVersionNumber: formData.get("expectedVersionNumber") || undefined,
  });
  if ("error" in parsed) {
    return parsed.error;
  }
  const { planId, skillId, recommendedMethod, instructions, prompts, levels, descriptors, expectedVersionNumber } =
    parsed.data;
  const rubric = descriptors.map((descriptor, index) => ({
    level: Number(levels[index] ?? ""),
    descriptor,
  }));
  if (rubric.some((band) => !band.descriptor || Number.isNaN(band.level))) {
    return fail("validation", USER_MESSAGES.validation);
  }
  return runAction(async () => {
    const token = await sessionToken();
    if (expectedVersionNumber != null) {
      const current = await readPlan({ token, planId });
      if (!current) {
        return fail("not_found", USER_MESSAGES.not_found);
      }
      if (current.versionNumber !== expectedVersionNumber) {
        return fail("conflict", USER_MESSAGES.conflict);
      }
    }
    try {
      const plan = await editPlanSkill(token, planId, skillId, {
        recommendedMethod,
        instructions,
        assessmentItems: prompts.map((prompt) => ({ prompt })),
        rubric,
      });
      revalidatePath("/admin");
      return ok({ planId: plan.id }, "Skill saved.");
    } catch (error) {
      if (error instanceof PlanImmutable) {
        return fail("conflict", "An approved plan cannot be edited.");
      }
      throw error;
    }
  });
}

async function movePlan(
  formData: FormData,
  action: (token: string, planId: string) => Promise<{ id: string }>,
  failure: string,
): Promise<ActionResult> {
  return runAction(async () => {
    const parsed = parseForm(planIdsSchema, {
      assignmentId: text(formData, "assignmentId"),
      planId: text(formData, "planId"),
    });
    if ("error" in parsed) {
      redirect("/admin");
    }
    const { assignmentId, planId } = parsed.data;
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
  });
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
