# Woodpecker CI support and rename to woodpecker-telegram

## Overview

The user's CI is Woodpecker 3.21. The GitHub repo is now `alkk/woodpecker-telegram` (renamed from `alkk/forgejo-telegram`; the Forgejo direction is dropped). Today the plugin mostly works on Woodpecker through Woodpecker's Drone compatibility layer, but it reports `success` on failed pipelines and misses the start time and deploy target. Drone support must stay unchanged.

## Context (from discovery)

Verified in Woodpecker v3.21.0 source:
- `pipeline/runtime/step.go` `setStepEnv`: for **plugin** steps it calls `metadata.SetDroneEnviron` and *then* sets `CI_PIPELINE_STATUS` (`success`/`failure` from the runtime error) and `CI_PIPELINE_STARTED` (the workflow's actual start).
- `pipeline/frontend/metadata/drone_compatibility.go`: copies `CI_*` into `DRONE_*` for repo, commit, build number/event/link, tag and PR. It hardcodes `DRONE_BUILD_STATUS=success`. `DRONE_BUILD_STARTED` comes from the compile-time value and can be `"0"`. There is no `DRONE_STAGE_STARTED` (what the plugin reads) and no `DRONE_DEPLOY_TO`.
- `Container.IsPlugin()` (`pipeline/frontend/yaml/types/container.go`): a step is a plugin when it has no `commands`, `entrypoint` or `environment`; `settings:` doesn't affect that.
  - Non-plugin steps get no `DRONE_*` aliases, so most metadata is missing there. They still get the three native `CI_*` values this plan reads.
  - The broader non-plugin case is deferred to `docs/backlog/woodpecker-non-plugin-step-loses-metadata.md`, which is uncommitted and goes into the Task 2 commit.
- Settings arrive as `PLUGIN_*`. Lists of primitives are comma-joined, so `to:` lists work as on Drone.

Repo facts:
- urfave/cli v1 (`flagFromFileEnv` in the vendored `flag.go`) takes the **first env var that is set, even if empty**. `unsetEmptyEnv` in `main.go` exists to undo that.
- `App.Run` parses every flag before it calls the action, so any set-but-invalid numeric or boolean env var fails the parse.
- Stock Drone doesn't inject `CI_PIPELINE_STATUS` or the other `CI_PIPELINE_*` names.
- The template field is `{{repo.namespace}}`. DOCS.md documents `{{repo.owner}}`, which renders empty (checked with drone-template-lib).
- Dependencies are vendored. Never run `go mod tidy -modfile=go.tools.mod`.
- The Makefile's `build_linux_*` targets use the unset `DEPLOY_IMAGE`, so `-o` is a directory and Go names the binary after the module path's last element. Goreleaser names binaries from `.ProjectName`, which it takes from the git remote.
- `goreleaser check` already fails on this config (version 0 and deprecated fields). That's pre-existing and tracked in `docs/backlog/goreleaser-config-v0-deprecated-fields.md`; a snapshot still builds, with warnings.

## Solution Overview

The three env lists, which are the substance of the change:
- **Status:** `build.status` reads `CI_PIPELINE_STATUS,DRONE_BUILD_STATUS`. On Woodpecker this beats the hardcoded `success`; on Drone the first variable is unset, so behaviour is unchanged. We ruled out a `CI_SYSTEM_NAME=woodpecker` identity check: ordering alone is enough and adds no code path. A caller that sets a non-empty `CI_PIPELINE_STATUS` on Drone would now win; we accept that.
- **Start time:** `build.started` reads `DRONE_STAGE_STARTED,CI_PIPELINE_STARTED`. `DRONE_BUILD_STARTED` is deliberately left out: Woodpecker can set it to `"0"`, which parses and so isn't cleared by `unsetEmptyEnv`.
- **Deploy target:** `deploy.to` reads `DRONE_DEPLOY_TO,CI_PIPELINE_DEPLOY_TARGET`. Its help text currently says the value is only for promotion/rollback pipelines; reword it to cover Woodpecker deployments.

Also:
- `unsetEmptyEnv` additionally clears an empty `CI_PIPELINE_STATUS` and `CI_PIPELINE_STARTED`.
- No other `CI_*` fallbacks are added; see the backlog item. `plugin.go` is untouched.

Ruled out: Forgejo Actions support, Woodpecker-only mode (dropping `DRONE_*`), and Woodpecker plugin-index metadata.

## Tandem

| Task | Writer | Reviewer |
|---|---|---|
| 1 | claude | codex |
| 2 | claude | codex |
| 3 | claude | codex |

## Implementation Steps

### Task 1: Rename to woodpecker-telegram

Touches: module path, tools module, lint config, build (Makefile, Dockerfile, .gitignore), docs (README, DOCS.md, CLAUDE.md), the `plugin.go` comment example, and the git remote. This is the same mechanical rename as the forgejo one in commit 636f543. Replace `forgejo-telegram` with `woodpecker-telegram` in the renamed files. Leave `appleboy/drone-telegram` references, historical plan text, `plan.md` and test data alone.

- [ ] every `forgejo-telegram` reference in the renamed files is now `woodpecker-telegram`, including the Dockerfile `COPY`/`ENTRYPOINT` and OCI source label
- [ ] `origin` points at `git@github.com:alkk/woodpecker-telegram.git`
- [ ] fresh `make build_linux_amd64` and `make build_linux_arm64` each produce `release/linux/<arch>/woodpecker-telegram`, the path the Dockerfile copies
- [ ] after the remote change, `goreleaser build --snapshot --single-target --clean` names the binary `woodpecker-telegram-…`; its existing warnings are expected
- [ ] if a container engine is available, a local image build and a run of `--help` succeed
- [ ] `go vet ./...`, `go test ./...`, `make lint`, `make fmt-check`, `go mod verify` pass
- [ ] committed as `chore!: rename to woodpecker-telegram` with `Tandem-Task: 1`

### Task 2: Woodpecker status, start time and deploy target

Touches: CLI flags and env handling (`main.go`), tests (`main_test.go`), and the docs that describe the behaviour (README, DOCS.md, CLAUDE.md), plus the backlog items.

**Test seam:**
- Move the production `unsetEmptyEnv` key list and the flag list into a `newApp()` that `main()` calls after loading the env file.
- The test builds a fresh app per case, replaces `Action` with a callback that reads `build.status`, `build.started` and `deploy.to` from the `*cli.Context`, and calls `Run` with only the program name.
- So the regression test goes through the real cleanup list: forgetting to add a key to production fails it.
- No `Plugin` extraction is needed.

**Test cases:** a serial table test in `main_test.go`.
- **Setup:** each case first clears every `DRONE_*`, `GITHUB_*` and `CI_*` variable, plus the other parsed numeric/bool inputs (`*_MESSAGE_THREAD_ID`, `PLUGIN_DEBUG`, `DEBUG`, `GITHUB`, …). Unset and empty are kept as distinct states.
- **Woodpecker failure:** `DRONE_BUILD_STATUS=success`, `CI_PIPELINE_STATUS=failure`, `DRONE_BUILD_STARTED=0`, a real `CI_PIPELINE_STARTED` and `CI_PIPELINE_DEPLOY_TARGET` give `failure`, the real start time and the target.
- **Drone only:** `DRONE_*` values are used unchanged.
- **Precedence:** distinct `DRONE_STAGE_STARTED`/`CI_PIPELINE_STARTED` and `DRONE_DEPLOY_TO`/`CI_PIPELINE_DEPLOY_TARGET` values show that `DRONE_*` wins for those two.
- **Empty values:** an empty `CI_PIPELINE_STATUS` falls back to `DRONE_BUILD_STATUS`. An empty `CI_PIPELINE_STARTED` with no `DRONE_STAGE_STARTED` parses to `Started=0`.

**README:**
- A Woodpecker step replaces the Forgejo Actions section, which removes the wrong `github.event.compare` line. The step uses `settings:` with `from_secret` for `token`/`to` and `when: [{event: push, status: [success, failure]}]`.
- Notes:
  - Woodpecker withholds secrets from `pull_request` pipelines by default.
  - In a serial workflow the step goes last. With `depends_on`, list every step whose outcome the message should report.
  - Full metadata needs a plugin step, meaning no `commands`, `entrypoint` or `environment`.
  - The commit author is the forge username.
  - `build.finished` is not available.
- Intro: "Woodpecker CI plugin, also works with Drone". The Drone and Docker sections stay.

**DOCS.md:**
- Woodpecker framing.
- In the template reference, `repo.owner` becomes `repo.namespace`, and the `build.link` text changes from "in drone" to "in the CI".
- Add the deploy target field. Check the name it actually renders under (likely `build.deployTo`) before documenting it.

**CLAUDE.md:**
- Woodpecker framing, with the telegram-action note removed.
- State the three exact env lists above and the empty-value cleanup.
- State that the first set variable wins even if empty, and that Woodpecker's `DRONE_*` aliases exist only for plugin steps.
- Don't claim a general `DRONE_*` > `GITHUB_*` > `CI_*` order; only these three fields read `CI_*`.

- [ ] the three env lists, the cleanup keys and the `deploy.to` help text are as above; Drone results unchanged
- [ ] the table test covers the cases above through `newApp()`; `go test ./...` passes
- [ ] `make lint`, `make fmt-check` pass
- [ ] README, DOCS.md, CLAUDE.md updated as described
- [ ] committed with `Tandem-Task: 2`

⚠️ Both `docs/backlog/` items went in with the plan commit instead of Task 2, so the tree was clean for the tandem run.

### Task 3: Close the plan

- [ ] this plan moved to `docs/plans/completed/`

## Post-Completion

Both commits stay local. After they land, these are separate steps:
- **Publish:** push, then tag to start goreleaser and the image build. Make `ghcr.io/alkk/woodpecker-telegram` public after the first publish, since new packages start private.
- **Smoke test:** run the published image in a real Woodpecker pipeline that fails on purpose. The message must show the failure icon and status, and a link to the pipeline.
- **Tracked elsewhere:** issue #1 (lowercase `markdown` skips escaping), the non-plugin-step backlog item, and the GoReleaser config backlog item.
