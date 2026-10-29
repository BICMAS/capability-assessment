"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { fail, ok, type ActionErr, type ActionResult } from "@/lib/action-result";
import { USER_MESSAGES } from "@/lib/errors";
import { parseForm, text, texts } from "@/lib/form-data";
import { runAction } from "@/lib/run-action";
import {
  addRoleProfileSkill,
  answerClarification,
  assignRoleProfileApproval,
  commentOnRoleProfile,
  compareRoleProfileVersions,
  createCanonicalSkill,
  createNextRoleProfileDraft,
  createProfileRole,
  createRoleProfile,
  decideRoleProfileApproval,
  mergeRoleProfileSkills,
  publishRoleProfile,
  readRoleProfile,
  removeRoleProfileSkill,
  reorderRoleProfileSkills,
  resubmitRoleProfile,
  saveApprovalPath,
  saveRoleProfileSkillEdits,
  retryRoleProfileJob,
  RoleProfileApiError,
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
  const organogramId = text(formData, "organogramId");
  const positionId = text(formData, "positionId");
  const linked = organogramId !== "" || positionId !== "";
  if (linked && !referencePair(organogramId, positionId)) {
    redirect(`/admin/role-profiles/new?error=${encodeURIComponent("Choose a position from a saved organogram.")}`);
  }
  let profileId = "";
  let draftVersionId = "";
  let revision = 1;
  try {
    const role = await createProfileRole(token, parsed.data.name);
    const profile = await createRoleProfile(token, role.id);
    profileId = profile.id;
    draftVersionId = profile.version.id;
    revision = profile.version.revision;
  } catch (error) {
    const mapped = roleProfileFailure(error);
    redirect(`/admin/role-profiles/new?error=${encodeURIComponent(mapped?.message ?? "The role profile was not created.")}`);
  }
  if (linked) {
    try {
      await updateRoleProfileContext(token, draftVersionId, {
        expectedRevision: revision,
        purpose: "",
        responsibilities: "",
        businessOutcomes: "",
        decisionScope: "",
        industry: "",
        region: "",
        teamSize: null,
        reportingLevel: "",
        organogramId,
        positionId,
        businessUnit: "",
        subIndustry: "",
        organisationalContext: "",
        sources: [],
      });
    } catch {
      revalidatePath("/admin/role-profiles");
      redirect(`/admin/role-profiles/${encodeURIComponent(profileId)}/context?organogram=${encodeURIComponent(organogramId)}&position=${encodeURIComponent(positionId)}`);
    }
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
  businessUnit: z.string().max(2000),
  subIndustry: z.string().max(200),
  organisationalContext: z.string().max(4000),
  frameworkVersionId: z.string().max(80),
  jobDescription: z.string().max(20000),
});

function contextInput(data: z.infer<typeof contextSchema>, position: { organogramId: string; positionId: string }, sources: ContextInput["sources"]): ContextInput {
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
    organogramId: position.organogramId,
    positionId: position.positionId,
    businessUnit: data.businessUnit,
    subIndustry: data.subIndustry,
    organisationalContext: data.organisationalContext,
    frameworkVersionId: data.frameworkVersionId,
    sources,
  };
}

function referencePair(organogramId: string, positionId: string): boolean {
  return organogramId !== "" && positionId !== "" && /^[A-Za-z0-9-]{1,80}$/.test(organogramId) && /^[A-Za-z0-9-]{1,80}$/.test(positionId);
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
    businessUnit: text(formData, "businessUnit"),
    subIndustry: text(formData, "subIndustry"),
    organisationalContext: text(formData, "organisationalContext"),
    frameworkVersionId: text(formData, "frameworkVersionId"),
    jobDescription: text(formData, "jobDescription"),
  });
  if ("error" in parsed) {
    return parsed.error;
  }
  if (parsed.data.teamSize !== "" && !/^[1-9]\d*$/.test(parsed.data.teamSize)) {
    return fail("validation", USER_MESSAGES.validation, { teamSize: ["Enter a whole number of people."] });
  }
  const position = selectedPosition(formData);
  if ("error" in position) {
    return position.error;
  }
  const sources = collectedSources(formData, parsed.data.jobDescription);
  if ("error" in sources) {
    return sources.error;
  }
  return call(async () => {
    const token = await sessionTokenOrThrow();
    await updateRoleProfileContext(token, parsed.data.versionId, contextInput(parsed.data, position, sources.sources));
    return { profileId: parsed.data.profileId };
  }, "Context saved.");
}

function selectedPosition(formData: FormData): { organogramId: string; positionId: string } | { error: ActionErr } {
  let organogramId = text(formData, "savedOrganogramId");
  let positionId = text(formData, "savedPositionId");
  if (text(formData, "positionSelection") === "present") {
    const choice = text(formData, "positionChoice");
    if (choice === "") {
      organogramId = "";
      positionId = "";
    } else {
      const separator = choice.indexOf(":");
      organogramId = separator > 0 ? choice.slice(0, separator) : "";
      positionId = separator > 0 ? choice.slice(separator + 1) : "";
    }
  }
  if (organogramId === "" && positionId === "") {
    return { organogramId, positionId };
  }
  if (!referencePair(organogramId, positionId)) {
    return { error: fail("validation", USER_MESSAGES.validation, { positionChoice: ["Choose a position from the organogram."] }) };
  }
  return { organogramId, positionId };
}

function collectedSources(formData: FormData, jobDescription: string): { sources: ContextInput["sources"] } | { error: ActionErr } {
  const titles = formValues(formData, "sourceTitle");
  const uris = formValues(formData, "sourceUri");
  const kinds = formValues(formData, "sourceKind");
  const bodies = formValues(formData, "sourceBody");
  const origins = formValues(formData, "sourceOrigin");
  if (titles.length !== uris.length || titles.length !== kinds.length || titles.length !== bodies.length || titles.length !== origins.length) {
    return { error: fail("validation", USER_MESSAGES.validation, { sourceTitle: ["Each source needs a title and a reference."] }) };
  }
  const sources: ContextInput["sources"] = [];
  for (let index = 0; index < titles.length; index += 1) {
    const title = titles[index].trim();
    const referenceUri = uris[index].trim();
    if (title === "" && referenceUri === "") {
      continue;
    }
    if (title === "" || referenceUri === "") {
      return { error: fail("validation", USER_MESSAGES.validation, { sourceTitle: ["Each source needs a title and a reference."] }) };
    }
    const origin = origins[index].trim();
    if (origin !== "" && origin !== "user" && origin !== "imported" && origin !== "ai_suggested") {
      return { error: fail("validation", USER_MESSAGES.validation, { sourceOrigin: ["The source origin is not valid."] }) };
    }
    sources.push({
      title,
      referenceUri,
      kind: kinds[index] === "job_description" ? "job_description" : "reference",
      body: bodies[index].trim(),
      origin: origin === "" ? "user" : origin,
    });
  }
  const ownsJobDescription = formData.has("jobDescription");
  const kept = ownsJobDescription ? sources.filter((source) => source.referenceUri !== "entered:job-description") : sources;
  if (ownsJobDescription && jobDescription.trim() !== "") {
    kept.push({
      title: "Job description",
      referenceUri: "entered:job-description",
      kind: "job_description",
      body: jobDescription.trim(),
      origin: "user",
    });
  }
  const jobFileId = text(formData, "jobFileId");
  if (jobFileId !== "") {
    if (!/^[A-Za-z0-9-]{1,80}$/.test(jobFileId)) {
      return { error: fail("validation", USER_MESSAGES.validation, { jobFileId: ["The uploaded file could not be saved."] }) };
    }
    const referenceUri = `file:${jobFileId}`;
    if (!kept.some((source) => source.referenceUri === referenceUri)) {
      kept.push({
        title: text(formData, "jobFileName") || "Uploaded job description",
        referenceUri,
        kind: "job_description",
        origin: "imported",
      });
    }
  }
  return { sources: kept };
}

function formValues(formData: FormData, name: string): string[] {
  return formData.getAll(name).filter((value): value is string => typeof value === "string");
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
  const replaceExisting = text(formData, "replaceExisting") === "yes";
  if (replaceExisting && text(formData, "confirmReplace") !== "on") {
    return fail("validation", "Confirm replacement before generating again.");
  }
  return runAction(async () => {
    try {
      const job = await startRoleProfileGeneration(
        await sessionTokenOrThrow(),
        parsed.data.versionId,
        parsed.data.contextVersion,
        parsed.data.idempotencyKey,
        replaceExisting,
      );
      return ok({ profileId: parsed.data.profileId, jobId: job.id }, "Generation queued.");
    } catch (error) {
      if (error instanceof RoleProfileApiError && (error.status === 409 || error.serviceCode === "conflict")) {
        return fail(
          "conflict",
          replaceExisting
            ? "Generation did not replace this draft because a kept skill uses the same canonical skill."
            : "This draft already has skills. Confirm replacement before generating again.",
        );
      }
      const mapped = roleProfileFailure(error);
      if (mapped) {
        return mapped;
      }
      throw error;
    }
  });
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
  criteria: z.string().max(2000),
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
      criteria: parsed.data.criteria,
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

const approvalDecisions = ["approved", "rejected", "changes_requested"] as const;

export async function decideRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      assignmentId: z.string(),
      decision: z.enum(approvalDecisions),
      comment: z.string().max(4000),
      expectedRevision: z.coerce.number().int().min(1),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      assignmentId: text(formData, "assignmentId"),
      decision: text(formData, "decision"),
      comment: text(formData, "comment"),
      expectedRevision: text(formData, "expectedRevision"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  if (parsed.data.decision !== "approved" && parsed.data.comment.trim() === "") {
    return fail("validation", "Record the reason for this decision.");
  }
  const notice = parsed.data.decision === "approved" ? "Approval recorded." : parsed.data.decision === "rejected" ? "Rejection recorded." : "Changes requested.";
  return call(async () => {
    await decideRoleProfileApproval(await sessionTokenOrThrow(), parsed.data.versionId, {
      assignmentId: parsed.data.assignmentId,
      decision: parsed.data.decision,
      comment: parsed.data.comment,
      expectedRevision: parsed.data.expectedRevision,
    });
    return { profileId: parsed.data.profileId };
  }, notice);
}

export async function decideRoleProfileForm(formData: FormData): Promise<void> {
  const result = await decideRoleProfileResult(formData);
  redirect(location(text(formData, "profileId"), "review", result));
}

export async function resubmitRoleProfileResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  return revisionAction(formData, resubmitRoleProfile, "Resubmitted for review.");
}

export async function resubmitRoleProfileForm(formData: FormData): Promise<void> {
  const result = await resubmitRoleProfileResult(formData);
  redirect(location(text(formData, "profileId"), "review", result));
}

export async function assignApprovalForm(formData: FormData): Promise<void> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      assigneePersonId: z.string().min(1).max(80),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      assigneePersonId: text(formData, "assigneePersonId"),
    },
  );
  if ("error" in parsed) {
    redirect(location(text(formData, "profileId"), "review", parsed.error));
  }
  const result = await call(async () => {
    await assignRoleProfileApproval(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.assigneePersonId);
    return { profileId: parsed.data.profileId };
  }, "Reviewer assigned.");
  redirect(location(parsed.data.profileId, "review", result));
}

export async function saveApprovalPathForm(formData: FormData): Promise<void> {
  const preset = text(formData, "preset");
  const stages = preset === "leader"
    ? [{ stageKey: "leader_approval", label: "Business Leader Approval", requiredRole: "leader" as const, position: 1 }]
    : [
        { stageKey: "hr_review", label: "HR/L&D Review", requiredRole: "hr" as const, position: 1 },
        { stageKey: "manager_review", label: "Line Manager Review", requiredRole: "manager" as const, position: 2 },
        { stageKey: "leader_approval", label: "Business Leader Approval", requiredRole: "leader" as const, position: 3 },
      ];
  const result = await call(async () => {
    await saveApprovalPath(await sessionTokenOrThrow(), stages);
    return { profileId: "queue" };
  }, "Approval path saved.");
  const params = new URLSearchParams();
  if (!result.ok) {
    params.set("error", result.message);
  } else if (result.message) {
    params.set("notice", result.message);
  }
  redirect(`/admin/role-profiles/review${params.toString() ? `?${params.toString()}` : ""}`);
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
  redirect(location(text(formData, "profileId"), text(formData, "returnTo") === "skills" ? "" : "review", result));
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
    criteria: text(formData, "criteria"),
    provenance: text(formData, "provenance"),
  };
}

const reviewEditSchema = z.object({
  skillAssignmentId: z.string().min(1),
  definition: z.string().min(1, "Enter a definition.").max(4000),
  requiredProficiency: z.number().int().min(0).max(4),
  priority: z.enum(["required", "preferred", "optional"]),
  rationale: z.string().min(1, "Enter a rationale.").max(4000),
  displayName: z.string().max(160),
  category: z.enum(["technical", "leadership", "domain", "communication", "compliance"]),
  criteria: z.string().max(2000),
});

export async function saveSkillReviewResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
      edits: z.string().min(2, "There are no skill edits to save."),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      expectedRevision: text(formData, "expectedRevision"),
      edits: text(formData, "edits"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  let decoded: unknown;
  try {
    decoded = JSON.parse(parsed.data.edits);
  } catch {
    return fail("validation", USER_MESSAGES.validation);
  }
  const edits = z.array(reviewEditSchema).safeParse(decoded);
  if (!edits.success) {
    return fail("validation", USER_MESSAGES.validation);
  }
  return call(async () => {
    await saveRoleProfileSkillEdits(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.expectedRevision, edits.data);
    return { profileId: parsed.data.profileId };
  }, "Changes saved.");
}

export async function saveSkillReviewForm(formData: FormData): Promise<void> {
  const result = await saveSkillReviewResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function saveSkillOrderResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
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
  const ids = texts(formData, "skillAssignmentId");
  if (ids.length < 2) {
    return fail("validation", USER_MESSAGES.validation);
  }
  return call(async () => {
    await reorderRoleProfileSkills(await sessionTokenOrThrow(), parsed.data.versionId, parsed.data.expectedRevision, ids);
    return { profileId: parsed.data.profileId };
  }, "Order saved.");
}

export async function saveSkillOrderForm(formData: FormData): Promise<void> {
  const result = await saveSkillOrderResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}

export async function mergeSkillsResult(formData: FormData): Promise<ActionResult<{ profileId: string }>> {
  if (text(formData, "confirm") !== "yes") {
    return fail("validation", USER_MESSAGES.validation, { confirm: ["Confirm the merge before removing a skill."] });
  }
  const parsed = parseForm(
    z.object({
      profileId: z.string().min(1),
      versionId: z.string().min(1),
      expectedRevision: z.coerce.number().int().min(1),
      keepId: z.string().min(1),
      removeId: z.string().min(1),
    }),
    {
      profileId: text(formData, "profileId"),
      versionId: text(formData, "versionId"),
      expectedRevision: text(formData, "expectedRevision"),
      keepId: text(formData, "keepId"),
      removeId: text(formData, "removeId"),
    },
  );
  if ("error" in parsed) {
    return parsed.error;
  }
  if (parsed.data.keepId === parsed.data.removeId) {
    return fail("validation", "Choose two different skills to merge.");
  }
  return call(async () => {
    await mergeRoleProfileSkills(
      await sessionTokenOrThrow(),
      parsed.data.versionId,
      parsed.data.expectedRevision,
      parsed.data.keepId,
      parsed.data.removeId,
    );
    return { profileId: parsed.data.profileId };
  }, "Duplicate merged.");
}

export async function mergeSkillsForm(formData: FormData): Promise<void> {
  const result = await mergeSkillsResult(formData);
  redirect(location(text(formData, "profileId"), "", result));
}
