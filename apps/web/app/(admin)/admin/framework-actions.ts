"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import {
  addCapability,
  addIndicator,
  addRequirement,
  addRole,
  createDraft,
  createNextDraft,
  approveVersion,
  decideProposal,
  publishVersion,
  saveIndicatorStatement,
  submitForReview,
  uploadImport,
} from "@/lib/capability-framework";
import { SESSION_COOKIE } from "@/lib/session";

async function sessionToken(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  return token;
}

export async function createFrameworkDraft(formData: FormData) {
  const name = text(formData, "frameworkName");
  if (!name) {
    return;
  }
  await createDraft(await sessionToken(), name);
  revalidatePath("/admin");
}

export async function addFrameworkRole(formData: FormData) {
  const versionId = text(formData, "versionId");
  const name = text(formData, "roleName");
  if (!versionId || !name) {
    return;
  }
  await addRole(await sessionToken(), versionId, name);
  revalidatePath("/admin");
}

export async function addFrameworkCapability(formData: FormData) {
  const versionId = text(formData, "versionId");
  const roleId = text(formData, "roleId");
  const name = text(formData, "capabilityName");
  const requiredLevel = Number(formData.get("requiredLevel"));
  if (!versionId || !roleId || !name || !Number.isInteger(requiredLevel)) {
    return;
  }
  const token = await sessionToken();
  const capability = await addCapability(token, versionId, name);
  await addRequirement(token, versionId, roleId, capability.id, requiredLevel);
  revalidatePath("/admin");
}

export async function saveFrameworkIndicator(formData: FormData) {
  const versionId = text(formData, "versionId");
  const capabilityId = text(formData, "capabilityId");
  const statement = text(formData, "indicator");
  const level = Number(formData.get("level"));
  if (!versionId || !capabilityId || !statement || !Number.isInteger(level)) {
    return;
  }
  await addIndicator(await sessionToken(), versionId, capabilityId, level, statement);
  revalidatePath("/admin");
}

export async function editPublishedFramework(formData: FormData) {
  const versionId = text(formData, "versionId");
  if (!versionId) {
    return;
  }
  await createNextDraft(await sessionToken(), versionId);
  revalidatePath("/admin");
}

export async function importFrameworkCsv(formData: FormData) {
  const file = formData.get("file");
  if (!(file instanceof File) || file.size === 0) {
    return;
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
}

export async function rejectProposal(formData: FormData) {
  await decide(formData, "rejected");
}

export async function acceptProposal(formData: FormData) {
  await decide(formData, "accepted");
}

export async function submitFrameworkForReview(formData: FormData) {
  const versionId = text(formData, "versionId");
  if (!versionId) {
    return;
  }
  const status = await submitForReview(await sessionToken(), versionId);
  revalidatePath("/admin");
  if (status === 409) {
    redirect(`/admin?review=blocked&action=submit&versionId=${encodeURIComponent(versionId)}`);
  }
  if (status !== 200) {
    redirect(`/admin?importError=${encodeURIComponent("Review could not be started.")}`);
  }
  redirect("/admin");
}

export async function saveNegotiation(formData: FormData) {
  const versionId = text(formData, "versionId");
  const indicatorId = text(formData, "indicatorId");
  const statement = text(formData, "statement");
  const frameworkName = text(formData, "frameworkName");
  const versionNumber = text(formData, "versionNumber");
  if (!versionId || !indicatorId || !statement) {
    return;
  }
  const status = await saveIndicatorStatement(await sessionToken(), versionId, indicatorId, statement);
  revalidatePath("/admin");
  if (status === 409) {
    redirect(
      `/admin?save=frozen&framework=${encodeURIComponent(frameworkName)}&version=${encodeURIComponent(versionNumber)}&versionId=${encodeURIComponent(versionId)}`,
    );
  }
  if (status !== 200) {
    redirect(`/admin?importError=${encodeURIComponent("Negotiation could not be saved.")}`);
  }
  redirect("/admin");
}

export async function approveFramework(formData: FormData) {
  const versionId = text(formData, "versionId");
  if (!versionId) {
    return;
  }
  const status = await approveVersion(await sessionToken(), versionId);
  revalidatePath("/admin");
  if (status === 409) {
    redirect(`/admin?review=blocked&action=approve&versionId=${encodeURIComponent(versionId)}`);
  }
  if (status !== 200) {
    redirect(`/admin?importError=${encodeURIComponent("The framework could not be approved.")}`);
  }
  redirect("/admin");
}

export async function publishFramework(formData: FormData) {
  const versionId = text(formData, "versionId");
  if (!versionId) {
    return;
  }
  const status = await publishVersion(await sessionToken(), versionId);
  revalidatePath("/admin");
  if (status === 409) {
    redirect(`/admin?review=blocked&action=publish&versionId=${encodeURIComponent(versionId)}`);
  }
  if (status !== 200) {
    redirect(`/admin?importError=${encodeURIComponent("The framework could not be published.")}`);
  }
  redirect("/admin");
}

async function decide(formData: FormData, status: "accepted" | "rejected") {
  const versionId = text(formData, "versionId");
  const proposalId = text(formData, "proposalId");
  if (!versionId || !proposalId) {
    return;
  }
  await decideProposal(await sessionToken(), versionId, proposalId, status);
  revalidatePath("/admin");
}

function text(formData: FormData, name: string): string {
  const value = formData.get(name);
  return typeof value === "string" ? value.trim() : "";
}
