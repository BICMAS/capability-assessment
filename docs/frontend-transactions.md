# Frontend mutation boundaries

## Capability + role requirement

Creating a capability and attaching a role requirement is **one business operation**.

The earlier UI called `addCapability` then `addRequirement`. If the second call failed, the capability existed without a requirement — a partial write the admin could not recover from cleanly.

**Owning service:** `capability-framework`

**Atomic API:** `POST /framework-versions/{versionId}/role-capabilities`

Body: `{ name, roleId, requiredLevel }`

The Go store (`AddRoleCapability`) creates the capability and requirement in one SQL transaction (memory store uses a single critical section). The Next.js server action `addFrameworkCapability` calls only this endpoint.

Do not reintroduce a frontend two-step for this operation.

## Other multi-call flows

| Flow | Boundary | Notes |
|------|----------|--------|
| CSV import | Service owns validation + create | UI surfaces missing columns / row errors only |
| Assignment batch | `assessment-plan` `assignPeople` | One request; per-person outcomes in the response |
| Plan skill edit | Single `editPlanSkill` | UI sends `expectedVersionNumber` to refuse stale overlays |
| Framework review / publish | Single status transition | `409` means a pending proposal blocks the transition |

## Safe errors

Server actions map failures to `ActionResult` codes (`validation`, `unauthenticated`, `forbidden`, `conflict`, `not_found`, `unavailable`, `unexpected`). User-visible strings come from `USER_MESSAGES` or known domain messages — never raw backend payloads or stack traces.
