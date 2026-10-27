"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { fail, ok, type ActionResult } from "@/lib/action-result";
import { USER_MESSAGES } from "@/lib/errors";
import { parseForm, text, texts } from "@/lib/form-data";
import { runAction } from "@/lib/run-action";
import {
  addRoleProfileSkill,
  answerClarification,
  commentOnRoleProfile,
  compareRoleProfileVersions,
  createCanonicalSkill,
  createNextRoleProfileDraft,
  createProfileRole,
  createRoleProfile,
  publishRoleProfile,
  readRoleProfile,
  removeRoleProfileSkill,
  reorderRoleProfileSkills,
  retryRoleProfileJob,
  roleProfileFailure,
  startClarification,
  startRoleProfileGeneration,
  submitRoleProfile,
  updateRoleProfileContext,
  updateRoleProfileSkill,
  type ContextInput,
  type RoleProfileComparison,
} from "@/lib/role-profile";
import { sessionTokenOrThrow } from "@/lib/session-token";

const reportingLevels = ["", "individual", "team_lead", "manager", "director", "executive"] as const;

function location(profileId: string, page: string, result: ActionResult<unknown>): string {
  const params = new URLSearchParams();
  if (!result.ok) {
    params.set("error", result.message);
  } else if (result.message) {
    params.set("notice", result.message);
  }
  const suffix = page ? `/${page}` : "";
  const query = params.toString();
  return `/admin/role-profiles/${encodeURIComponent(profileId)}${suffix}${query ? `?${query}` : ""}`;
}

async function call<T>(fn: () => Promise<T>, message: string): Promise<ActionResult<T>> {
  return runAction(async () => {
    try {
      const data = await fn();
      return ok(data, message);
    } catch (error) {
      const mapped = roleProfileFailure(error);
      if (mapped) {
        return mapped;
      }
      throw error;
    }
  });
}

export async function createRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({ name: z.string().min(1, "Enter a role name.").max(160) }),
    { name: text(formData, "name") },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    const token = await sessionTokenOrThrow();
    const role = await createProfileRole(token, parsed.data.name);
    const profile = await createRoleProfile(token, role.id);
    return { profileId: profile.id };
  }, "Draft created.");
}

export async function createRoleProfileForm(formData: FormData): Promise<void> {
  const parsed = parseForm(
    z.object({ name: z.string().min(1, "Enter a role name.").max(160) }),
    { name: text(formData, "name") },
  );
  if ("error" in parsed) {
    redirect(`/admin/role-profiles/new?error=${encodeURIComponent(parsed.error.message)}`);
  }
  const token = await sessionTokenOrThrow();
  let profileId = "";
  try {
    const role = await createProfileRole(token, parsed.data.name);
    const profile = await createRoleProfile(token, role.id);
    profileId = profile.id;
  } catch (error) {
    const mapped = roleProfileFailure(error);
    redirect(`/admin/role-profiles/new?error=${encodeURIComponent(mapped?.message ?? "The role profile was not created.")}`);
  }
  revalidatePath("/admin/role-profiles");
  redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}/context`);
}

export async function openRoleProfileForm(formData: FormData): Promise<void> {
  const profileId = text(formData, "profileId");
  if (!profileId) {
    redirect(`/admin/role-profiles?error=${encodeURIComponent("Enter a profile id.")}`);
  }
  try {
    await readRoleProfile(await sessionTokenOrThrow(), profileId);
  } catch (error) {
    const mapped = roleProfileFailure(error);
    redirect(`/admin/role-profiles?error=${encodeURIComponent(mapped?.message ?? USER_MESSAGES.unexpected)}`);
  }
  redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}`);
}

const contextSchema = z.object({
  profileId: z.string().min(1),
  versionId: z.string().min(1),
  expectedRevision: z.coerce.number().int().min(1),
  purpose: z.string().max(4000),
  responsibilities: z.string().max(4000),
  businessOutcomes: z.string().max(4000),
  decisionScope: z.string().max(4000),
  industry: z.string().max(200),
  region: z.string().max(200),
  teamSize: z.string(),
  reportingLevel: z.enum(reportingLevels),
});

function contextInput(data: z.infer<typeof contextSchema>, sources: Array<{ title: string; referenceUri: string }>): ContextInput {
  const teamSize = data.teamSize === "" ? null : Number(data.teamSize);
  return {
    expectedRevision: data.expectedRevision,
    purpose: data.purpose,
    responsibilities: data.responsibilities,
    businessOutcomes: data.businessOutcomes,
    decisionScope: data.decisionScope,
    industry: data.industry,
    region: data.region,
    teamSize: teamSize != null && Number.isInteger(teamSize) ? teamSize : null,
    reportingLevel: data.reportingLevel,
    sources,
  };
}

export async function saveRoleContextResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(contextSchema, {
    profileId: text(formData, "profileId"),
    versionId: text(formData, "versionId"),
    expectedRevision: text(formData, "expectedRevision"),
    purpose: text(formData, "purpose"),
    responsibilities: text(formData, "responsibilities"),
    businessOutcomes: text(formData, "businessOutcomes"),
    decisionScope: text(formData, "decisionScope"),
    industry: text(formData, "industry"),
    region: text(formData, "region"),
    teamSize: text(formData, "teamSize"),
    reportingLevel: text(formData, "reportingLevel"),
  });
  if ("error" in parsed) {
    return parsed.error;
  }
  if (parsed.data.teamSize !== "" && !/^[1-9]\d*$/.test(parsed.data.teamSize)) {
    return fail("validation", USER_MESSAGES.validation, { teamSize: ["Enter a whole number of people."] });
  }
  const titles = texts(formData, "sourceTitle");
  const uris = texts(formData, "sourceUri");
  if (titles.length !== uris.length) {
    return fail("validation", USER_MESSAGES.validation, { sourceTitle: ["Each source needs a title and a reference."] });
  }
  const sources = titles.map((title, index) => ({ title, referenceUri: uris[index] ?? "" }));
  return call(async () => {
    const token = await sessionTokenOrThrow();
    await updateRoleProfileContext(token, parsed.data.versionId, contextInput(parsed.data, sources));
    return { profileId: parsed.data.profileId };
  }, "Context saved.");
}

export async function saveRoleContext(formData: FormData): Promise<void> {
  const result = await saveRoleContextResult(formData);
  redirect(location(text(formData, "profileId"), "context", result));
}

export async function saveRoleSource(formData: FormData): Promise<void> {
  const result = await saveRoleContextResult(formData);
  redirect(location(text(formData, "profileId"), "sources", result));
}

export async function startClarificationForm(formData: FormData): Promise<void> {
  const profileId = text(formData, "profileId");
  const versionId = text(formData, "versionId");
  const result = await call(async () => {
    await startClarification(await sessionTokenOrThrow(), versionId);
    return { profileId };
  }, "Clarification saved.");
  redirect(location(profileId, "clarify", result));
}

export async function answerClarificationResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      questionId: z.string().min(1, "Choose a question."),
      value: z.string().max(4000),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      questionId: text(formData, "questionId"),
      value: text(formData, "value"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await answerClarification(await sessionTokenOrThrow(), parsed.data.versionId, [
      { questionId: parsed.data.questionId, value: parsed.data.value },
    ]);
    return { profileId: parsed.data.profileId };
  }, "Answer saved.");
}

export async function answerClarificationForm(formData: FormData): Promise<void> {
  const result = await answerClarificationResult(formData);
  redirect(location(text(formData, "profileId"), "clarify", result));
}

export async function startGenerationResult(formData: FormData): Promise<ActionResult<{ profileId: string; jobId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      contextVersion: z.coerce.number().int().min(1, "Choose a context version."),
      idempotencyKey: z.string().min(8).max(80),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      contextVersion: text(formData, "contextVersion"),
      idempotencyKey: text(formData, "idempotencyKey"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    const job = await startRoleProfileGeneration(
      await sessionTokenOrThrow(),
      parsed.data.versionId,
      parsed.data.contextVersion,
      parsed.data.idempotencyKey,
    );
    return { profileId: parsed.data.profileId, jobId: job.id };
  }, "Generation queued.");
}

export async function startGenerationForm(formData: FormData): Promise<void> {
  const result = await startGenerationResult(formData);
  const profileId = text(formData, "profileId");
  if (!result.ok || !result.data) {
    redirect(location(profileId, "generation", result));
  }
  redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}/generation?job=${encodeURIComponent(result.data.jobId)}`);
}

export async function retryGenerationResult(formData: FormData): Promise<ActionResult<{ profileId: string; jobId: string }>> {
  const parsed = parseForm(
    z.object({ profileId: z.string().min(1), jobId: z.string().min(1) }),
    { profileId: text(formData, "profileId"), jobId: text(formData, "jobId") },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    const job = await retryRoleProfileJob(await sessionTokenOrThrow(), parsed.data.jobId);
    return { profileId: parsed.data.profileId, jobId: job.id };
  }, "Generation queued again.");
}

export async function retryGenerationForm(formData: FormData): Promise<void> {
  const result = await retryGenerationResult(formData);
  const profileId = text(formData, "profileId");
  const jobId = result.ok && result.data ? result.data.jobId : text(formData, "jobId");
  const params = new URLSearchParams({ job: jobId });
  if (!result.ok) {
    params.set("error", result.message);
  }
  redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}/generation?${params}`);
}

const skillSchema = z.object({
  profileId: z.string().min(1),
  versionId: z.string().min(1),
  expectedRevision: z.coerce.number().int().min(1),
  skillId: z.string().min(1, "Enter a canonical skill id."),
  category: z.string().min(1, "Enter the skill category."),
  definition: z.string().min(1, "Enter a definition.").max(4000),
  requiredProficiency: z.string().regex(/^\d+$/, "Enter a proficiency level."),
  priority: z.string().min(1, "Enter a priority."),
  rationale: z.string().min(1, "Enter a rationale.").max(4000),
  provenance: z.string().min(1, "Enter a provenance."),
});

export async function addSkillResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(skillSchema, skillFields(formData));
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await addRoleProfileSkill(await sessionTokenOrThrow(), parsed.data.versionId, {
      expectedRevision: parsed.data.expectedRevision,
      skillId: parsed.data.skillId,
      category: parsed.data.category,
      definition: parsed.data.definition,
      requiredProficiency: Number(parsed.data.requiredProficiency),
      priority: parsed.data.priority,
      rationale: parsed.data.rationale,
      provenance: parsed.data.provenance,
    });
    return { profileId: parsed.data.profileId };
  }, "Skill added.");
}

export async function addSkillForm(formData: FormData): Promise<void> {
  const result = await addSkillResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function createCanonicalSkillResult(formData: FormData): Promise<ActionResult<{ skillId: string }>> {
  const parsed = parseForm(
    z.object({
      name: z.string().min(1, "Enter a skill name.").max(160),
      category: z.string().min(1, "Enter a category."),
      definition: z.string().min(1, "Enter a definition.").max(4000),
    }),
    { name: text(formData, "name"), category: text(formData, "category"), definition: text(formData, "definition") },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    const skill = await createCanonicalSkill(await sessionTokenOrThrow(), parsed.data);
    return { skillId: skill.id };
  }, "Canonical skill created.");
}

export async function createCanonicalSkillForm(formData: FormData): Promise<void> {
  const profileId = text(formData, "profileId");
  const result = await createCanonicalSkillResult(formData);
  if (!result.ok || !result.data) {
    redirect(location(profileId, "", result));
  }
  const params = new URLSearchParams({
    skillId: result.data.skillId,
    category: text(formData, "category"),
    notice: result.message ?? "Canonical skill created.",
  });
  redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}?${params}`);
}

export async function updateSkillResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      skillAssignmentId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
      definition: z.string().min(1, "Enter a definition.").max(4000),
      requiredProficiency: z.string().regex(/^\d+$/, "Enter a proficiency level."),
      priority: z.string().min(1, "Enter a priority."),
      rationale: z.string().min(1, "Enter a rationale.").max(4000),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      skillAssignmentId: text(formData, "skillAssignmentId"),
      expectedRevision: text(formData, "expectedRevision"),
      definition: text(formData, "definition"),
      requiredProficiency: text(formData, "requiredProficiency"),
      priority: text(formData, "priority"),
      rationale: text(formData, "rationale"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await updateRoleProfileSkill(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.skillAssignmentId, {
      expectedRevision: parsed.data.expectedRevision,
      definition: parsed.data.definition,
      requiredProficiency: Number(parsed.data.requiredProficiency),
      priority: parsed.data.priority,
      rationale: parsed.data.rationale,
    });
    return { profileId: parsed.data.profileId };
  }, "Skill saved.");
}

export async function updateSkillForm(formData: FormData): Promise<void> {
  const result = await updateSkillResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function removeSkillResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  if (text(formData, "confirm") !== "yes") {
    return fail("validation", USER_MESSAGES.validation, { confirm: ["Confirm removal before deleting a skill."] });
  }
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      skillAssignmentId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      skillAssignmentId: text(formData, "skillAssignmentId"),
      expectedRevision: text(formData, "expectedRevision"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await removeRoleProfileSkill(
      await sessionTokenOrThrow(),
      parsed.data.versionId,
      parsed.data.skillAssignmentId,
      parsed.data.expectedRevision,
    );
    return { profileId: parsed.data.profileId };
  }, "Skill removed.");
}

export async function removeSkillForm(formData: FormData): Promise<void> {
  const result = await removeSkillResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function reorderSkillResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
      moveId: z.string().min(1),
      direction: z.enum(["up", "down"]),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      expectedRevision: text(formData, "expectedRevision"),
      moveId: text(formData, "moveId"),
      direction: text(formData, "direction"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  const ids = texts(formData, "skillAssignmentId");
  const index = ids.indexOf(parsed.data.moveId);
  const swap = parsed.data.direction === "up" ? index - 1 : index + 1;
  if (index < 0 || swap < 0 || swap >= ids.length) {
    return fail("validation", USER_MESSAGES.validation);
  }
  const next = [...ids];
  const current = next[index];
  const other = next[swap];
  if (!current || !other) {
    return fail("validation", USER_MESSAGES.validation);
  }
  next[index] = other;
  next[swap] = current;
  return call(async () => {
    await reorderRoleProfileSkills(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.expectedRevision, next);
    return { profileId: parsed.data.profileId };
  }, "Order saved.");
}

export async function reorderSkillForm(formData: FormData): Promise<void> {
  const result = await reorderSkillResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

async function revisionAction(
  formData: FormData,
  run: (token: string, versionId: string, expectedRevision: number) => Promise<unknown>,
  message: string,
): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      expectedRevision: text(formData, "expectedRevision"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await run(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.expectedRevision);
    return { profileId: parsed.data.profileId };
  }, message);
}

export async function submitRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  return revisionAction(formData, submitRoleProfile, "Submitted for review.");
}

export async function submitRoleProfileForm(formData: FormData): Promise<void> {
  const result = await submitRoleProfileResult(formData);
  redirect(location(text(formData, "profileId"), "review", result));
}

export async function publishRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  return revisionAction(formData, publishRoleProfile, "Published.");
}

export async function publishRoleProfileForm(formData: FormData): Promise<void> {
  const result = await publishRoleProfileResult(formData);
  redirect(location(text(formData, "profileId"), "review", result));
}

export async function commentOnRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      body: z.string().min(1, "Enter a comment.").max(4000),
    }),
    { profileId: text(formData, "profileId"), versionId: text(formData, "versionId"), body: text(formData, "body") },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    await commentOnRoleProfile(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.body);
    return { profileId: parsed.data.profileId };
  }, "Comment saved.");
}

export async function commentOnRoleProfileForm(formData: FormData): Promise<void> {
  const result = await commentOnRoleProfileResult(formData);
  redirect(location(text(formData, "profileId"), "review", result));
}

export async function nextDraftResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  return revisionAction(formData, createNextRoleProfileDraft, "New draft created.");
}

export async function nextDraftForm(formData: FormData): Promise<void> {
  const result = await nextDraftResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function compareRoleProfilesResult(formData: FormData): Promise<ActionResult<RoleProfileComparison>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      from: z.string().min(1, "Choose the earlier version."),
      to: z.string().min(1, "Choose the later version."),
    }),
    { profileId: text(formData, "profileId"), from: text(formData, "from"), to: text(formData, "to") },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  return call(async () => {
    const token = await sessionTokenOrThrow();
    return compareRoleProfileVersions(token, parsed.data.profileId, parsed.data.from, parsed.data.to);
  }, "Comparison loaded.");
}

function skillFields(formData: FormData) {
  return {
    profileId: text(formData, "profileId"),
    versionId: text(formData, "versionId"),
    expectedRevision: text(formData, "expectedRevision"),
    skillId: text(formData, "skillId"),
    category: text(formData, "category"),
    definition: text(formData, "definition"),
    requiredProficiency: text(formData, "requiredProficiency"),
    priority: text(formData, "priority"),
    rationale: text(formData, "rationale"),
    provenance: text(formData, "provenance"),
  };
}
