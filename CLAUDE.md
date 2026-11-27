# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Woodpecker CI plugin that sends Telegram notifications; it still works as a Drone plugin. Single `main` package: `main.go` (CLI flags, `newApp()`), `plugin.go` (all logic). The `INPUT_*`/`GITHUB_*` env aliases are inherited from upstream, which also backed a GitHub Action.

## Fork

- This checkout is `alkk/woodpecker-telegram` (`origin`), a fork of `appleboy/drone-telegram`. No `upstream` remote is configured; add one (`git remote add upstream git@github.com:appleboy/drone-telegram.git`) to pull upstream changes.
- Bugs go to GitHub issues on `alkk/woodpecker-telegram`. Issue/PR numbers in commit messages before the fork (`#149`, ...) and in `plan.md` refer to upstream repos.
- The Go module path is `github.com/alkk/woodpecker-telegram`; the tools module is `github.com/alkk/woodpecker-telegram/tools` (`go.tools.mod`), and the `gci` import prefix in `.golangci.yml` must match the module path. Everything user-facing (README, DOCS.md, Dockerfile labels, image names) is branded `alkk`.
- The only published image is `ghcr.io/alkk/woodpecker-telegram` (`.github/workflows/docker.yml`); there is no Docker Hub image.
- `LICENSE` keeps the upstream MIT copyright, as the license requires.

## Commands

```sh
make build        # bin/woodpecker-telegram
make test         # go test -v -cover ./... (writes coverage.txt)
make lint         # golangci-lint via go.tools.mod
make fmt          # golangci-lint fmt (gci, gofumpt, golines)
make fmt-check    # what CI runs; must pass
go test -run TestEscapeMarkdownContexts -v .   # single test
```

Dependencies are vendored: after changing `go.mod`, run `go mod tidy && go mod vendor` and commit `vendor/`. Dev tools (golangci-lint) are pinned in `go.tools.mod`, a separate module that `vendor/` does not cover, so the Makefile runs them with `go run -modfile=go.tools.mod -mod=mod <pkg>`; `go tool` fails here because it ignores `-mod`. Never run `go mod tidy -modfile=go.tools.mod`, it pulls the app's dependencies into the tools module. CI also runs hadolint on `docker/Dockerfile`.

## Tests

- Tests calling `skipIfNoTelegramSecrets(t)` send real messages and only run when `TELEGRAM_TOKEN` and `TELEGRAM_TO` are set.
- Two tests run without that guard. `TestBotError` attempts `getMe` with a dummy token and accepts any error, including a connection failure, so it also passes offline. `TestSendMessage` contacts Telegram whenever `TELEGRAM_TOKEN` is set (its recipient list has hard-coded entries) and sends real messages if the token and `TELEGRAM_TO` are valid.
- Env-alias tests (`TestNewAppWoodpeckerEnv`) run the real `newApp()` with its `Action` swapped for a callback, after `clearCIEnv` unsets every variable the flags read. Keep that pattern, so the production cleanup list and flag lists are what's tested.
- Testify is used; `testifylint` is enabled, so use `require` for fatal checks and assertion forms it accepts.
- Media fixtures live in `tests/`.

## Architecture

- **Config sources**: most flags in `main.go` read several env var aliases — `PLUGIN_*` (Drone), `TELEGRAM_*`, `INPUT_*` (GitHub Actions), plus `DRONE_*`/`GITHUB_*` build metadata. `PLUGIN_ENV_FILE` and `/run/drone/env` are loaded via godotenv before parsing. urfave/cli takes the **first alias that is set, even if empty**. `unsetEmptyEnv` (called by `newApp()`) clears empty values for its listed keys. Mostly these are numeric flags' aliases, because urfave/cli can't parse `""` as an int, so add new numeric flags there. The one string exception is `CI_PIPELINE_STATUS`, which is cleared so an empty value can't hide `DRONE_BUILD_STATUS`. Clearing changes first-set semantics, so don't add other string aliases without that reason.
- **Woodpecker**: for plugin steps (no `commands`, `entrypoint` or `environment`), Woodpecker copies its `CI_*` metadata into `DRONE_*` aliases, so most flags need no `CI_*` names. Only three fields read `CI_*`:
  - `build.status`: `CI_PIPELINE_STATUS,DRONE_BUILD_STATUS`. Woodpecker hardcodes `DRONE_BUILD_STATUS=success`; Drone never sets `CI_PIPELINE_STATUS`.
  - `build.started`: `DRONE_STAGE_STARTED,CI_PIPELINE_STARTED`. Not `DRONE_BUILD_STARTED`, which Woodpecker may set to `0`.
  - `deploy.to`: `DRONE_DEPLOY_TO,CI_PIPELINE_DEPLOY_TARGET`.

  Non-plugin steps get no `DRONE_*` aliases; see `docs/backlog/woodpecker-non-plugin-step-loses-metadata.md`.
- **Exec flow** (`Plugin.Exec`): token/`to` check → return early if `skip_file` exists (before any Telegram call; a `Stat` error other than not-exist fails) → pick message source (`MessageFile` > `Message` > default `Plugin.Message()`, for which `Exec` forces the Markdown format) → merge template vars (`template_vars_file` overrides inline `template_vars`) → build bot (optional SOCKS5) → `parseTo` resolves recipients → `renderMessages` → send text then each media type to every recipient. The first send error aborts.
- **Recipients**: `to` entries are `chat_id` or `chat_id:email`. The email form only sends if it matches the commit author email; with `only_match_email`, matched entries replace the plain ones.
- **Rendering order matters**: `renderMessages` runs the drone-template-lib (Handlebars/raymond) template first, then `html.UnescapeString`, then (Markdown only) `escapeMarkdownOne`. Escaping happens on the final text, never on template fields, so a variable can safely appear in both plain text and code/links.
- **Markdown escaping** targets Telegram's legacy `Markdown` parse mode, not MarkdownV2. `escapeMarkdownOne` escapes `_` in ordinary text but keeps it inside complete inline code, triple-backtick blocks, `[label](url)` links and bare http(s) URLs; existing `\_` stays as is. `markdownDelimiters` pre-indexes matching delimiters in one pass so unmatched brackets/backticks don't cause rescans. User-facing behavior is documented in README.md "Markdown messages" — keep it in sync.
- `Send` masks the bot token in returned errors.

## Docs

- `DOCS.md` is the plugin's settings reference; update it when adding or changing settings. The plugins.drone.io Telegram listing is upstream's, not this fork's.
- `plan.md` is the (Chinese-language) working plan for the Markdown underscore escaping fix (appleboy/telegram-action#30).
- Releases: goreleaser (`.goreleaser.yaml`) builds the binaries from the committed `vendor/` (no tidy hook, so a release doesn't depend on the module proxy); `docker/Dockerfile` copies a prebuilt `release/<os>/<arch>/woodpecker-telegram`.
- Dependabot keeps `go.mod` and `vendor/` in sync (it runs `go mod vendor` itself) but does not see `go.tools.mod`; bump golangci-lint by hand.
