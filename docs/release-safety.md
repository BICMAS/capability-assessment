# Release safety: migrations and rollout

## Principles

1. Prefer **additive** schema changes (new tables, new nullable columns, new indexes).
2. Application code must tolerate the previous schema during a rolling deploy.
3. Never run destructive migrations (`DROP`, irreversible `ALTER`, data deletion) as part of an automatic production deploy.
4. Domain services migrate their **own** database on process start via goose (`goose.Up`). That path is for additive ups only.

## Deployment sequence

1. **Land additive migrations** in the release that introduces them.
2. **Deploy application binaries/images** that understand both old and new schema shapes (expand phase).
3. Verify readiness (`GET /ready`) and critical workflows.
4. Only in a **later, manually approved** change: backfill, tighten constraints, or remove obsolete columns (contract phase).
5. Rollback = redeploy the previous application revision. Additive migrations stay; do not auto-run `goose.Down` in production.

## Rollback

| Situation | Action |
| --- | --- |
| Bad app revision, schema still compatible | Redeploy previous image/tag. Leave databases as-is. |
| Migration bug discovered before cutover | Stop the rollout. Fix forward with a new additive migration. |
| Destructive change required | Human-operated change window with backup/restore plan. Not CI, not auto-migrate. |

## What CI does not do

- CI integration tests create ephemeral databases and run `goose.Up` only.
- CI does not connect to production databases.
- CI does not apply `DOWN` migrations.

## Local / staging migrate

Each service reads `DATABASE_URL_<SERVICE>` (or `DATABASE_URL` override). Starting the service applies pending **up** migrations for that service’s embedded SQL only.

`services/capability-framework/migrations/00004_outbox.sql` only creates `outbox_events`. Leave that table in place on rollback; pending rows are published when a compatible process starts. Recovery steps are in `docs/transactional-outbox.md`.
