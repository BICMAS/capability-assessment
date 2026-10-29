"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { proficiencyPercentage } from "@/features/role-profiles/gap-scale";
import { recordAttempt } from "@/lib/generated/assessment-gap/levels";
import { SESSION_COOKIE } from "@/lib/session";

export async function saveSelfAssessment(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const personId = text(formData, "personId");
  const frameworkVersionId = text(formData, "frameworkVersionId");
  const capabilityCode = text(formData, "capabilityCode");
  const evidence = text(formData, "evidence");
  const assessed = Number(text(formData, "assessedProficiency"));
  const required = Number(text(formData, "requiredLevel"));
  const percentage = proficiencyPercentage(assessed);
  if (!personId || !frameworkVersionId || !capabilityCode || !evidence || percentage === null || !Number.isInteger(required) || required < 0 || required > 4) {
    redirect("/employee?assessmentError=Enter+a+proficiency+from+0+to+4+and+the+evidence.");
  }
  try {
    await recordAttempt(token, {
      personId,
      frameworkVersionId,
      capabilityCode,
      source: "self_assessment",
      percentage,
      evidenceIds: [evidence],
      requiredLevel: required,
    });
  } catch {
    redirect("/employee?assessmentError=The+self-assessment+was+not+stored.");
  }
  revalidatePath("/employee");
  redirect("/employee?assessmentNotice=Self-assessment+stored.+It+is+not+included+in+the+calculated+level.");
}

function text(formData: FormData, key: string): string {
  const value = formData.get(key);
  return typeof value === "string" ? value.trim() : "";
}
