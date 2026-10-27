export type Settled<T> =
  | { ok: true; value: T }
  | { ok: false; error: string };

export async function settle<T>(promise: Promise<T>, label: string): Promise<Settled<T>> {
  try {
    return { ok: true, value: await promise };
  } catch {
    return { ok: false, error: `${label} is unavailable right now.` };
  }
}
