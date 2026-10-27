"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { fail, ok, type ActionResult } from "@/lib/action-result";
import {
  addIndicator,
  addRole,
  addRoleCapability,
  createDraft,
  createNextDraft,
  approveVersion,
  decideProposal,
  publishVersion,
  saveIndicatorStatement,
  submitForReview,
  uploadImport,
} from "@/lib/capability-framework";
import { USER_MESSAGES } from "@/lib/errors";
import { parseForm, text } from "@/lib/form-data";
import { asFormAction } from "@/lib/form-action";
import { runAction } from "@/lib/run-action";
import { sessionToken } from "@/lib/session-token";

const versionIdSchema = z.object({
  versionId: z.string().min(1, "Choose a framework version."),
});

const draftSchema = z.object({
  frameworkName: z.string().min(1, "Enter a framework name.").max(200),
});

const roleSchema = z.object({
  versionId: z.string().min(1),
  roleName: z.string().min(1, "Enter a role name.").max(120),
});

const capabilitySchema = z.object({
  versionId: z.string().min(1),
  roleId: z.string().min(1, "Choose a role."),
  capabilityName: z.string().min(1, "Enter a capability name.").max(120),
  requiredLevel: z.coerce.number().int().min(0).max(4),
});

const indicatorSchema = z.object({
  versionId: z.string().min(1),
  capabilityId: z.string().min(1),
  indicator: z.string().min(1, "Enter an indicator.").max(2000),
  level: z.coerce.number().int().min(0).max(4),
});

const negotiationSchema = z.object({
  versionId: z.string().min(1),
  indicatorId: z.string().min(1),
  statement: z.string().min(1, "Enter a statement.").max(2000),
  frameworkName: z.string().optional(),
  versionNumber: z.string().optional(),
});

const proposalSchema = z.object({
  versionId: z.string().min(1),
  proposalId: z.string().min(1),
});

export async function createFrameworkDraft(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(draftSchema, {
      frameworkName: text(formData, "frameworkName"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    await createDraft(await sessionToken(), parsed.data.frameworkName);
    revalidatePath("/admin");
    return ok(undefined, "Draft created.");
  });
}

export async function addFrameworkRole(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(roleSchema, {
      versionId: text(formData, "versionId"),
      roleName: text(formData, "roleName"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    await addRole(await sessionToken(), parsed.data.versionId, parsed.data.roleName);
    revalidatePath("/admin");
    return ok(undefined, "Role added.");
  });
}

/** Uses the atomic Go role-capabilities endpoint — not a frontend two-step. */
export async function addFrameworkCapability(formData: FormData): Promise<void> {
  await addFrameworkCapabilityResult(formData);
}

export async function addFrameworkCapabilityResult(formData: FormData): Promise<ActionResult> {
  return runAction(async () => {
    const parsed = parseForm(capabilitySchema, {
      versionId: text(formData, "versionId"),
      roleId: text(formData, "roleId"),
      capabilityName: text(formData, "capabilityName"),
      requiredLevel: formData.get("requiredLevel"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    const { versionId, roleId, capabilityName, requiredLevel } = parsed.data;
    await addRoleCapability(await sessionToken(), versionId, roleId, capabilityName, requiredLevel);
    revalidatePath("/admin");
    return ok(undefined, "Capability added.");
  });
}

export async function saveFrameworkIndicator(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(indicatorSchema, {
      versionId: text(formData, "versionId"),
      capabilityId: text(formData, "capabilityId"),
      indicator: text(formData, "indicator"),
      level: formData.get("level"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    const { versionId, capabilityId, indicator, level } = parsed.data;
    await addIndicator(await sessionToken(), versionId, capabilityId, level, indicator);
    revalidatePath("/admin");
    return ok(undefined, "Indicator saved.");
  });
}

export async function editPublishedFramework(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(versionIdSchema, { versionId: text(formData, "versionId") });
    if ("error" in parsed) {
      return parsed.error;
    }
    await createNextDraft(await sessionToken(), parsed.data.versionId);
    revalidatePath("/admin");
    return ok(undefined, "New draft created.");
  });
}

export async function importFrameworkCsv(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const file = formData.get("file");
    if (!(file instanceof File) || file.size === 0) {
      return fail("validation", "Choose a CSV file to import.");
    }
    const outcome = await uploadImport(await sessionToken(), file);
    revalidatePath("/admin");
    if (outcome.missingColumns.length > 0) {
      redirect(`/admin?importError=${encodeURIComponent(`Missing columns: ${outcome.missingColumns.join(", ")}`)}`);
    }
    if (!outcome.created && outcome.rowErrors.length > 0) {
      const detail = outcome.rowErrors.map((rowError) => `Row ${rowError.row}: ${rowError.message}`).join(" ");
      redirect(`/admin?importError=${encodeURIComponent(detail)}`);
    }
    if (!outcome.created) {
      redirect(`/admin?importError=${encodeURIComponent("The CSV could not be imported.")}`);
    }
    return ok(undefined, "CSV imported.");
  });
}

export async function rejectProposal(formData: FormData): Promise<void> {
  await decide(formData, "rejected");
}

export async function acceptProposal(formData: FormData): Promise<void> {
  await decide(formData, "accepted");
}

export async function submitFrameworkForReview(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(versionIdSchema, { versionId: text(formData, "versionId") });
    if ("error" in parsed) {
      return parsed.error;
    }
    const { versionId } = parsed.data;
    const status = await submitForReview(await sessionToken(), versionId);
    revalidatePath("/admin");
    if (status === 409) {
      redirect(`/admin?review=blocked&action=submit&versionId=${encodeURIComponent(versionId)}`);
    }
    if (status !== 200) {
      return fail("unavailable", USER_MESSAGES.unavailable);
    }
    redirect("/admin");
  });
}

export async function saveNegotiation(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(negotiationSchema, {
      versionId: text(formData, "versionId"),
      indicatorId: text(formData, "indicatorId"),
      statement: text(formData, "statement"),
      frameworkName: text(formData, "frameworkName"),
      versionNumber: text(formData, "versionNumber"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    const { versionId, indicatorId, statement, frameworkName, versionNumber } = parsed.data;
    const status = await saveIndicatorStatement(await sessionToken(), versionId, indicatorId, statement);
    revalidatePath("/admin");
    if (status === 409) {
      redirect(
        `/admin?save=frozen&framework=${encodeURIComponent(frameworkName ?? "")}&version=${encodeURIComponent(versionNumber ?? "")}&versionId=${encodeURIComponent(versionId)}`,
      );
    }
    if (status !== 200) {
      return fail("conflict", USER_MESSAGES.conflict);
    }
    redirect("/admin");
  });
}

export async function approveFramework(formData: FormData): Promise<void> {
  await runLifecycle(formData, approveVersion, "approve", "The framework could not be approved.");
}

export async function publishFramework(formData: FormData): Promise<void> {
  await runLifecycle(formData, publishVersion, "publish", "The framework could not be published.");
}

async function runLifecycle(
  formData: FormData,
  action: (token: string, versionId: string) => Promise<number>,
  name: string,
  failure: string,
): Promise<ActionResult> {
  return runAction(async () => {
    const parsed = parseForm(versionIdSchema, { versionId: text(formData, "versionId") });
    if ("error" in parsed) {
      return parsed.error;
    }
    const { versionId } = parsed.data;
    const status = await action(await sessionToken(), versionId);
    revalidatePath("/admin");
    if (status === 409) {
      redirect(`/admin?review=blocked&action=${name}&versionId=${encodeURIComponent(versionId)}`);
    }
    if (status !== 200) {
      return fail("unavailable", failure);
    }
    redirect("/admin");
  });
}

async function decide(formData: FormData, status: "accepted" | "rejected"): Promise<ActionResult> {
  return runAction(async () => {
    const parsed = parseForm(proposalSchema, {
      versionId: text(formData, "versionId"),
      proposalId: text(formData, "proposalId"),
    });
    if ("error" in parsed) {
      return parsed.error;
    }
    await decideProposal(await sessionToken(), parsed.data.versionId, parsed.data.proposalId, status);
    revalidatePath("/admin");
    return ok(undefined, status === "accepted" ? "Proposal accepted." : "Proposal rejected.");
  });
}
