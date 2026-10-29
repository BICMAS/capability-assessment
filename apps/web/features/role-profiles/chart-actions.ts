"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { fail, ok, type ActionResult } from "@/lib/action-result";
import { createOrganogram, createPosition, deletePosition, publishOrganogram, updatePosition, type PositionInput } from "@/lib/org-directory";
import { sessionTokenOrThrow } from "@/lib/session-token";

export async function createOrganogramForm(formData: FormData) {
  const name = String(formData.get("name") ?? "").trim();
  if (!name) {
    redirect("/admin/role-profiles?error=" + encodeURIComponent("Enter an organogram name."));
  }
  let createdId = "";
  try {
    const created = await createOrganogram(await sessionTokenOrThrow(), name);
    createdId = created.id;
  } catch {
    redirect("/admin/role-profiles?error=" + encodeURIComponent("The organogram could not be saved."));
  }
  redirect("/admin/role-profiles?chart=" + encodeURIComponent(createdId) + "&notice=" + encodeURIComponent("Organogram saved."));
}

export async function addPositionAction(organogramId: string, input: PositionInput): Promise<ActionResult> {
  if (input.title.trim() === "") {
    return fail("validation", "Enter a position title.");
  }
  try {
    await createPosition(await sessionTokenOrThrow(), organogramId, input);
  } catch (error) {
    return fail("validation", error instanceof Error ? error.message : "The position could not be added.");
  }
  revalidatePath("/admin/role-profiles");
  return ok();
}

export async function savePositionAction(organogramId: string, positionId: string, input: PositionInput): Promise<ActionResult> {
  try {
    await updatePosition(await sessionTokenOrThrow(), organogramId, positionId, input);
  } catch (error) {
    return fail("validation", error instanceof Error ? error.message : "The position could not be saved.");
  }
  revalidatePath("/admin/role-profiles");
  return ok();
}

export async function removePositionAction(organogramId: string, positionId: string): Promise<ActionResult> {
  try {
    await deletePosition(await sessionTokenOrThrow(), organogramId, positionId);
  } catch (error) {
    return fail("validation", error instanceof Error ? error.message : "The position could not be removed.");
  }
  revalidatePath("/admin/role-profiles");
  return ok();
}

export async function publishOrganogramAction(organogramId: string): Promise<ActionResult> {
  try {
    await publishOrganogram(await sessionTokenOrThrow(), organogramId);
  } catch (error) {
    return fail("validation", error instanceof Error ? error.message : "The organogram could not be published.");
  }
  revalidatePath("/admin/role-profiles");
  return ok();
}
