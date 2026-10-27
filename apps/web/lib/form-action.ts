import type { ActionResult } from "@/lib/action-result";

/** Next.js `<form action>` expects `Promise<void>`; discard typed results after handling. */
export async function asFormAction(run: () => Promise<ActionResult<unknown>>): Promise<void> {
  await run();
}
