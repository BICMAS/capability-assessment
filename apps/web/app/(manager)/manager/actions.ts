"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { recordAttempt } from "@/lib/generated/assessment-gap/levels";
import { confirmAssessmentSession } from "@/lib/generated/assessment-plan/plans";
import { listPeople } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

export async function saveManagerAssessment(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const personId = text(formData, "personId");
  const frameworkVersionId = text(formData, "frameworkVersionId");
  const capabilityCode = text(formData, "capabilityCode");
  const evidence = text(formData, "evidence");
  const percentage = Number(text(formData, "percentage"));
  const requiredLevel = Number(text(formData, "requiredLevel"));
  if (!personId || !frameworkVersionId || !capabilityCode || !evidence) {
    return;
  }
  if (!Number.isInteger(percentage) || percentage < 0 || percentage > 100) {
    return;
  }
  if (!Number.isFinite(requiredLevel)) {
    return;
  }
  const people = await listPeople(token);
  if (!people.some((person) => person.id === personId)) {
    redirect("/manager");
  }
  await recordAttempt(token, {
    personId,
    frameworkVersionId,
    capabilityCode,
    source: "manager_assessment",
    percentage,
    evidenceIds: [evidence],
    requiredLevel,
  });
  revalidatePath("/manager");
  revalidatePath(`/manager/people/${personId}`);
}

export async function confirmProposal(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const personId = text(formData, "personId");
  const sessionId = text(formData, "sessionId");
  if (!personId || !sessionId) {
    return;
  }
  const people = await listPeople(token);
  if (!people.some((person) => person.id === personId)) {
    redirect("/manager");
  }
  await confirmAssessmentSession(token, sessionId);
  revalidatePath("/manager");
  revalidatePath(`/manager/people/${personId}`);
  revalidatePath("/employee");
}

function text(formData: FormData, key: string): string {
  const value = formData.get(key);
  return typeof value === "string" ? value.trim() : "";
}
