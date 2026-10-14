OpenAPI specifications for the Go services live here.

- `audit.yaml` — append-only audit records
- `notify.yaml` — in-app inbox
- `org-directory.yaml` — departments, people, and role assignments
- `capability-framework.yaml` — versioned capability frameworks
- `assessment-gap.yaml` — assessment attempts, calculated levels, and manager overrides
- `development.yaml` — course mappings, pathways, and the learning completion webhook
- `assessment-plan.yaml` — role assignments and assessment plans

The Next.js inbox client is generated from `notify.yaml`. The directory client is generated from `org-directory.yaml`. The assessment client is generated from `assessment-gap.yaml`. The pathway client is generated from `development.yaml`. The assessment plan client is written by hand from `assessment-plan.yaml`.
