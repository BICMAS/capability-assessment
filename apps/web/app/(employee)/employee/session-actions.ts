"use server";

import { revalidatePath } from "next/cache";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import {
  listAssessmentSessions,
  saveAssessmentAnswers,
  startAssessmentSession,
  submitAssessmentSession,
} from "@/lib/generated/assessment-plan/plans";
import { SESSION_COOKIE } from "@/lib/session";

export async function submitSessionAnswers(formData: FormData) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const sessionId = text(formData, "sessionId");
  if (!sessionId) {
    redirect("/employee?sessionError=Choose%20a%20session.");
  }
  let message = "";
  try {
    const sessions = await listAssessmentSessions(token);
    const session = sessions.find((item) => item.id === sessionId && item.personId.length > 0);
    if (!session || session.status === "submitted") {
      message = "This session cannot be submitted.";
    } else {
      if (session.status === "assigned") {
        await startAssessmentSession(token, session.id);
      }
      const answers = session.skills.flatMap((skill) =>
        skill.items.map((item) => ({ itemId: item.id, answer: text(formData, `answer.${item.id}`) })),
      );
      if (answers.length === 0 || answers.some((answer) => answer.answer.length === 0)) {
        message = "Write an answer for each item.";
      } else {
        await saveAssessmentAnswers(token, session.id, answers);
        await submitAssessmentSession(token, session.id);
      }
    }
  } catch {
    message = "The answers could not be submitted.";
  }
  if (message) {
    redirect(`/employee?sessionError=${encodeURIComponent(message)}`);
  }
  revalidatePath("/employee");
  redirect("/employee");
}

function text(formData: FormData, name: string): string {
  const value = formData.get(name);
  return typeof value === "string" ? value.trim() : "";
}
