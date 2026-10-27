import { type ActionErrorCode, type ActionResult } from "@/lib/action-result";
import { toActionError } from "@/lib/errors";

export async function runAction<T = undefined>(
  fn: () => Promise<ActionResult<T>>,
  fallback: ActionErrorCode = "unexpected",
): Promise<ActionResult<T>> {
  try {
    return await fn();
  } catch (error) {
    return toActionError(error, fallback);
  }
}
