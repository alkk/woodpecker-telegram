---
worth: yes
where: main.go
added: 2026-11-27
---
# Woodpecker non-plugin steps get no build metadata

Woodpecker 3.21 sets the `DRONE_*` compat variables (via `SetDroneEnviron`) only for steps it classifies as plugins. As soon as a woodpecker-telegram step adds `environment:` (e.g. `HTTPS_PROXY`) or overrides `entrypoint`, it stops being a plugin. See `pipeline/frontend/yaml/types/container.go` at tag v3.21.0. Only `CI_*` remain, so repo, build number, build link, author and the rest come out empty.

Fix: append `CI_*` fallbacks after the `DRONE_*`/`GITHUB_*` names in the `main.go` flag `EnvVar` lists. The list: `CI_REPO`, `CI_REPO_OWNER`, `CI_REPO_NAME`, the `CI_COMMIT_*` set, `CI_PIPELINE_NUMBER`/`EVENT`/`URL`/`FORGE_URL`/`AVATAR`, `CI_COMMIT_TAG` and `CI_COMMIT_PULL_REQUEST`. Drone sets `CI_REPO_NAME` to the full `owner/name` slug, so `DRONE_REPO_NAME` must stay first. `CI_PIPELINE_NUMBER` needs `unsetEmptyEnv`.

Deferred during the Woodpecker-support design. Status, start time and deploy target were fixed then. The README documents the settings-only step as the supported form.
