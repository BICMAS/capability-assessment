"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { z } from "zod";

import { fail, ok, type ActionResult } from "@/lib/action-result";
import {
  VersionNotPublished,
  assignPeople,
  type AssignmentResult,
} from "@/lib/generated/assessment-plan/plans";
import { USER_MESSAGES } from "@/lib/errors";
import { parseForm, text, texts } from "@/lib/form-data";
import { asFormAction } from "@/lib/form-action";
import { runAction } from "@/lib/run-action";
import { sessionToken } from "@/lib/session-token";

const assignmentSchema = z.object({
  frameworkVersionId: z.string().min(1, "Choose a published version."),
  roleId: z.string().min(1, "Choose a role."),
  personIds: z.array(z.string().min(1)).min(1, "Choose at least one person."),
  mismatchPersonIds: z.array(z.string()),
  exceptionReason: z.string().max(280),
});

export async function confirmAssignment(formData: FormData): Promise<void> {
  return asFormAction(async () => {
    const parsed = parseForm(assignmentSchema, {
      frameworkVersionId: text(formData, "frameworkVersionId"),
      roleId: text(formData, "roleId"),
      personIds: texts(formData, "personId"),
      mismatchPersonIds: texts(formData, "mismatchPersonId"),
      exceptionReason: text(formData, "exceptionReason"),
    });
    if ("error" in parsed) {
      redirect(`/admin?assignmentError=${encodeURIComponent(USER_MESSAGES.validation)}`);
    }
    const { frameworkVersionId, roleId, personIds, mismatchPersonIds, exceptionReason } = parsed.data;
    const mismatched = new Set(mismatchPersonIds);
    if (mismatched.size > 0 && exceptionReason.length === 0) {
      redirect(`/admin?assignmentError=${encodeURIComponent("A title mismatch needs an exception reason.")}`);
    }

    let results: AssignmentResult[];
    try {
      const batch = await assignPeople(await sessionToken(), {
        frameworkVersionId,
        roleId,
        people: personIds.map((personId) => ({
          personId,
          exceptionReason: mismatched.has(personId) ? exceptionReason : undefined,
        })),
      });
      results = batch.results;
    } catch (error) {
      const message =
        error instanceof VersionNotPublished ? "The framework version is not published." : USER_MESSAGES.unavailable;
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
  });
}

export async function confirmAssignmentResult(formData: FormData): Promise<ActionResult<{ results: AssignmentResult[] }>> {
  const parsed = parseForm(assignmentSchema, {
    frameworkVersionId: text(formData, "frameworkVersionId"),
    roleId: text(formData, "roleId"),
    personIds: texts(formData, "personId"),
    mismatchPersonIds: texts(formData, "mismatchPersonId"),
    exceptionReason: text(formData, "exceptionReason"),
  });
  if ("error" in parsed) {
    return parsed.error;
  }
  const { frameworkVersionId, roleId, personIds, mismatchPersonIds, exceptionReason } = parsed.data;
  const mismatched = new Set(mismatchPersonIds);
  if (mismatched.size > 0 && exceptionReason.length === 0) {
    return fail("validation", "A title mismatch needs an exception reason.", {
      exceptionReason: ["Enter a reason for the title mismatch."],
    });
  }
  return runAction(async () => {
    try {
      const batch = await assignPeople(await sessionToken(), {
        frameworkVersionId,
        roleId,
        people: personIds.map((personId) => ({
          personId,
          exceptionReason: mismatched.has(personId) ? exceptionReason : undefined,
        })),
      });
      revalidatePath("/admin");
      return ok({ results: batch.results }, "Assignment saved.");
    } catch (error) {
      if (error instanceof VersionNotPublished) {
        return fail("conflict", "The framework version is not published.");
      }
      throw error;
    }
  });
}
