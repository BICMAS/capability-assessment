import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { AppError } from "@/lib/errors";
import { SESSION_COOKIE } from "@/lib/session";

export async function sessionToken(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  return token;
}

export async function sessionTokenOrThrow(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    throw new AppError("unauthenticated");
  }
  return token;
}
