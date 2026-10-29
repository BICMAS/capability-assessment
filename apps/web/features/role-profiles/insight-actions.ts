"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { proficiencyPercentage } from "@/features/role-profiles/gap-scale";
import { createLearningRecommendation, createLearningResource, createPracticeSession, type PracticeReply } from "@/lib/capability-insights";
import { recordAttempt } from "@/lib/generated/assessment-gap/levels";
import { SESSION_COOKIE } from "@/lib/session";

export type PracticeResult = { feedback: string; recorded: boolean; voice: string; error: string };

export async function saveProfileAssessment(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const profileId = text(formData, "profileId");
  if (!token) {
    redirect("/");
  }
  const personId = text(formData, "personId");
  const versionId = text(formData, "profileVersionId");
  const skill = text(formData, "skillId").split(":");
  const skillId = skill[0] ?? "";
  const method = text(formData, "method");
  const evidence = text(formData, "evidence");
  const assessed = Number(text(formData, "assessedProficiency"));
  const required = Number(skill[1] ?? "");
  const percentage = proficiencyPercentage(assessed);
  if (!profileId || !personId || !versionId || !skillId || !evidence || percentage === null || !Number.isInteger(required) || required < 0 || required > 4) {
    redirect(insightPath(profileId, "Enter the employee, skill, proficiency from 0 to 4, and evidence."));
  }
  if (method !== "manager_assessment" && method !== "self_assessment") {
    redirect(insightPath(profileId, "Choose manager assessment or employee self-assessment."));
  }
  try {
    await recordAttempt(token, {
      personId,
      frameworkVersionId: versionId,
      capabilityCode: skillId,
      source: method,
      percentage,
      evidenceIds: [evidence],
      requiredLevel: required,
    });
  } catch {
    redirect(insightPath(profileId, "The assessment was not stored. Check that this person can be assessed with the selected method."));
  }
  revalidatePath(`/admin/role-profiles/${profileId}/insights`);
  redirect(insightPath(profileId, "", "Assessment stored. The assessor is the signed-in person. The status is provisional until the assessment service verifies it. Self-assessment is stored and is not part of the calculated level."));
}

export async function saveLearningResource(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const profileId = text(formData, "profileId");
  if (!token) {
    redirect("/");
  }
  const title = text(formData, "title");
  const skillId = text(formData, "skillId");
  const reference = text(formData, "reference");
  const rationale = text(formData, "rationale");
  const targetLevel = Number(text(formData, "targetLevel"));
  if (!profileId || !title || !skillId || !reference || !rationale || !Number.isInteger(targetLevel) || targetLevel < 0 || targetLevel > 4) {
    redirect(insightPath(profileId, "Enter a title, skill, target level from 0 to 4, reference, and rationale."));
  }
  try {
    await createLearningResource(token, { title, skillId, targetLevel, reference, rationale });
  } catch {
    redirect(insightPath(profileId, "The learning resource was not stored. No learning system is confirming enrolment."));
  }
  revalidatePath(`/admin/role-profiles/${profileId}/insights`);
  redirect(insightPath(profileId, "", "Learning resource saved. It is not marked enrolled, started, or completed."));
}

export async function recommendForGap(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const profileId = text(formData, "profileId");
  if (!token) {
    redirect("/");
  }
  const gap = text(formData, "gap").split(":");
  const personId = gap[0] ?? "";
  const capabilityId = gap[1] ?? "";
  const targetLevel = Number(gap[2] ?? "");
  const state = text(formData, "state");
  if (!profileId || !personId || !capabilityId || state !== "below" || !Number.isInteger(targetLevel)) {
    redirect(insightPath(profileId, "A recommendation needs a confirmed gap. A missing assessment is not a confirmed gap."));
  }
  try {
    await createLearningRecommendation(token, { personId, capabilityId, targetLevel, state });
  } catch {
    redirect(insightPath(profileId, "No matching learning resource or course mapping was found. Enrolment was not created."));
  }
  revalidatePath(`/admin/role-profiles/${profileId}/insights`);
  redirect(insightPath(profileId, "", "Recommendation saved with status recommended. Enrolment is not confirmed."));
}

export async function practiseScenario(_previous: PracticeResult, formData: FormData): Promise<PracticeResult> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    return { feedback: "", recorded: false, voice: "unconfigured", error: "Sign in before starting a practice session." };
  }
  const personId = text(formData, "personId");
  const scenarioKey = text(formData, "scenarioKey");
  const message = text(formData, "message");
  const record = formData.get("record") === "yes";
  if (!personId || !scenarioKey || !message) {
    return { feedback: "", recorded: false, voice: "unconfigured", error: "Choose a scenario and enter what the person would say." };
  }
  let reply: PracticeReply;
  try {
    reply = await createPracticeSession(token, { personId, scenarioKey, message, record });
  } catch {
    return { feedback: "", recorded: false, voice: "unconfigured", error: "The text practice prototype is unavailable. No voice service is configured." };
  }
  return { feedback: reply.feedback, recorded: reply.recorded, voice: reply.voice, error: "" };
}

function insightPath(profileId: string, error: string, notice = ""): string {
  const params = new URLSearchParams();
  if (error) {
    params.set("error", error);
  }
  if (notice) {
    params.set("notice", notice);
  }
  const query = params.toString();
  return `/admin/role-profiles/${encodeURIComponent(profileId)}/insights${query ? `?${query}` : ""}`;
}

function text(formData: FormData, key: string): string {
  const value = formData.get(key);
  return typeof value === "string" ? value.trim() : "";
}
