# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Drone CI plugin (also used as the backend of the `appleboy/telegram-action` GitHub Action) that sends Telegram notifications. Single `main` package: `main.go` (CLI flags), `plugin.go` (all logic).

## Fork

- This checkout is `alkk/drone-telegram` (`origin`), a fork of `appleboy/drone-telegram`. No `upstream` remote is configured; add one (`git remote add upstream git@github.com:appleboy/drone-telegram.git`) to pull upstream changes.
- Bugs go to GitHub issues on `alkk/drone-telegram`. Issue/PR numbers in commit messages before the fork (`#149`, ...) and in `plan.md` refer to upstream repos.
- The Go module path is `github.com/alkk/drone-telegram`; it also appears in `go.tools.mod` and the `gci` import prefix in `.golangci.yml`. Everything user-facing (README, DOCS.md, Dockerfile labels, image names) is branded `alkk`.
- The only published image is `ghcr.io/alkk/drone-telegram` (`.github/workflows/docker.yml`); there is no Docker Hub image.
- `LICENSE` keeps the upstream MIT copyright, as the license requires.

## Commands

```sh
make build        # bin/drone-telegram
make test         # go test -v -cover ./... (writes coverage.txt)
make lint         # golangci-lint via go.tools.mod
make fmt          # golangci-lint fmt (gci, gofumpt, golines)
make fmt-check    # what CI runs; must pass
go test -run TestEscapeMarkdownContexts -v .   # single test
```

Dev tools (golangci-lint etc.) are pinned in `go.tools.mod` and run with `go tool -modfile=go.tools.mod <tool>`, not installed globally. CI also runs hadolint on `docker/Dockerfile`.

## Tests

- Tests calling `skipIfNoTelegramSecrets(t)` send real messages and only run when `TELEGRAM_TOKEN` and `TELEGRAM_TO` are set. Other tests run offline; `TestSendMessage`/`TestBotError` call `Exec` but only assert an error.
- Testify is used; `testifylint` is enabled, so use `require` for fatal checks and assertion forms it accepts.
- Media fixtures live in `tests/`.

## Architecture

- **Config sources**: every flag in `main.go` reads several env var aliases — `PLUGIN_*` (Drone), `TELEGRAM_*`, `INPUT_*` (GitHub Actions), plus `DRONE_*`/`GITHUB_*` build metadata. `PLUGIN_ENV_FILE` and `/run/drone/env` are loaded via godotenv before parsing. `unsetEmptyEnv` exists because GitHub Actions exports empty `INPUT_*` values that urfave/cli can't parse as ints — add new numeric flags to that list.
- **Exec flow** (`Plugin.Exec`): pick message source (`MessageFile` > `Message` > default `Plugin.Message()`, which forces Markdown) → merge template vars (`template_vars_file` overrides inline `template_vars`) → build bot (optional SOCKS5) → `parseTo` resolves recipients → `renderMessages` → send text then each media type to every recipient. The first send error aborts.
- **Recipients**: `to` entries are `chat_id` or `chat_id:email`. The email form only sends if it matches the commit author email; with `only_match_email`, matched entries replace the plain ones.
- **Rendering order matters**: `renderMessages` runs the drone-template-lib (Handlebars/raymond) template first, then `html.UnescapeString`, then (Markdown only) `escapeMarkdownOne`. Escaping happens on the final text, never on template fields, so a variable can safely appear in both plain text and code/links.
- **Markdown escaping** targets Telegram's legacy `Markdown` parse mode, not MarkdownV2. `escapeMarkdownOne` escapes `_` in ordinary text but keeps it inside complete inline code, triple-backtick blocks, `[label](url)` links and bare http(s) URLs; existing `\_` stays as is. `markdownDelimiters` pre-indexes matching delimiters in one pass so unmatched brackets/backticks don't cause rescans. User-facing behavior is documented in README.md "Markdown messages" — keep it in sync.
- `Send` masks the bot token in returned errors.

## Docs

- `DOCS.md` is the plugin's settings reference (published on plugins.drone.io); update it when adding or changing settings.
- `plan.md` is the (Chinese-language) working plan for the Markdown underscore escaping fix (appleboy/telegram-action#30).
- Releases: goreleaser (`.goreleaser.yaml`) builds the binaries; `docker/Dockerfile` copies a prebuilt `release/<os>/<arch>/drone-telegram`.
