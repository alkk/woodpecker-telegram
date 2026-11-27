# forgejo-telegram

![logo](./images/logo.png)

[![GitHub release](https://img.shields.io/github/v/release/alkk/forgejo-telegram)](https://github.com/alkk/forgejo-telegram/releases)

Send Telegram notifications from CI pipelines. Runs as a container step in
[Forgejo Actions](https://forgejo.org/docs/latest/user/actions/), as a
[Drone](https://github.com/harness/drone) plugin, or as a standalone binary.

Forked from [appleboy/drone-telegram](https://github.com/appleboy/drone-telegram).

## Features

* Text messages in Telegram `Markdown` or `HTML` format
* Photo, document, audio, voice, video and sticker messages
* Location and venue messages
* Messages to a forum topic via `message_thread_id`
* Handlebars message templates with build metadata and extra `template_vars` / `template_vars_file`
* Message loaded from a file with `message_file`
* Per-recipient filtering by commit author email (`only_match_email`)
* Silent messages (`disable_notification`) and no link preview (`disable_web_page_preview`)
* SOCKS5 proxy
* All settings loaded from an env file (`PLUGIN_ENV_FILE`)

## Configuration

Every setting is an environment variable, usually `PLUGIN_<NAME>` or
`TELEGRAM_<NAME>`: `PLUGIN_TOKEN`, `PLUGIN_TO`, `PLUGIN_MESSAGE` and so on.
Drone turns `settings:` into `PLUGIN_*` variables for you; elsewhere, set them
directly.

`token` is the bot token from [@BotFather](https://t.me/BotFather). `to` is a
comma-separated list of chat ids. An entry written as `chat_id:email` only
receives the message when `email` matches the commit author.

Build metadata for the default message and templates (`build.status`,
`commit.author`, ...) comes from `DRONE_*` variables. See [DOCS.md](DOCS.md) for
every setting, template variable and helper.

## Usage

### Forgejo Actions

Forgejo doesn't set `DRONE_*` variables, so map the build metadata from
workflow contexts. Use `env:`, not `with:`, to pass settings to a `docker://`
step.

```yaml
- name: Telegram notification
  if: always()
  uses: docker://ghcr.io/alkk/forgejo-telegram:latest
  env:
    PLUGIN_TOKEN: ${{ secrets.TELEGRAM_TOKEN }}
    PLUGIN_TO: ${{ secrets.TELEGRAM_TO }}
    DRONE_BUILD_STATUS: ${{ job.status }}
    DRONE_BUILD_NUMBER: ${{ github.run_number }}
    DRONE_BUILD_EVENT: ${{ github.event_name }}
    DRONE_BUILD_LINK: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_number }}
    DRONE_COMMIT_BRANCH: ${{ github.ref_name }}
    DRONE_COMMIT_AUTHOR: ${{ github.event.head_commit.author.name }}
    DRONE_COMMIT_AUTHOR_EMAIL: ${{ github.event.head_commit.author.email }}
    DRONE_COMMIT_MESSAGE: ${{ github.event.head_commit.message }}
    DRONE_COMMIT_LINK: ${{ github.event.compare }}
```

The repository name and commit SHA come from Forgejo's own `GITHUB_REPOSITORY`
and `GITHUB_SHA`. `head_commit` exists only in push event payloads; on other
events the commit fields stay empty.

### Drone

```yaml
- name: send telegram notification
  image: ghcr.io/alkk/forgejo-telegram
  settings:
    token:
      from_secret: telegram_token
    to:
      from_secret: telegram_to
    message: >
      {{#success build.status}}
        build {{build.number}} succeeded. Good job.
      {{else}}
        build {{build.number}} failed. Fix me please.
      {{/success}}
```

### Docker

```sh
docker run --rm \
  -e PLUGIN_TOKEN=xxxxxxx \
  -e PLUGIN_TO=xxxxxxx \
  -e PLUGIN_MESSAGE=test \
  ghcr.io/alkk/forgejo-telegram
```

Media settings take local file paths, so mount the working directory:

```sh
docker run --rm \
  -e PLUGIN_TOKEN=xxxxxxx \
  -e PLUGIN_TO=xxxxxxx \
  -e PLUGIN_MESSAGE="build {{build.number}} of {{repo.name}}: {{build.status}}" \
  -e PLUGIN_PHOTO=tests/github.png \
  -e PLUGIN_DOCUMENT=tests/gophercolor.png \
  -e PLUGIN_LOCATION="24.9163213 121.1424972" \
  -e PLUGIN_VENUE="24.9163213 121.1424972 title address" \
  -e DRONE_REPO_NAME=forgejo-telegram \
  -e DRONE_COMMIT_SHA=e5e82b5eb3737205c25955dcc3dcacc839b7be52 \
  -e DRONE_COMMIT_BRANCH=master \
  -e DRONE_COMMIT_AUTHOR=alkk \
  -e DRONE_BUILD_NUMBER=1 \
  -e DRONE_BUILD_STATUS=success \
  -v "$(pwd)":"$(pwd)" \
  -w "$(pwd)" \
  ghcr.io/alkk/forgejo-telegram
```

To keep everything in one file, pass `-e PLUGIN_ENV_FILE=path/to/file` instead.

## Markdown escaping

With `format: Markdown` (the default), underscores in ordinary text are escaped:
`user_name` is sent as `user\_name` so Telegram doesn't start italics.
Underscores stay as they are inside complete inline code, triple-backtick
blocks, Markdown links (label and URL) and bare HTTP(S) URLs:

```yaml
format: Markdown
message: |
  Built release_name
  Directory: `_site`
  [_site](https://example.com/_site)
```

Escaping runs after template rendering, so a variable can appear both in text
and inside code or links. Existing `\_` stays as written. Unclosed code or link
syntax falls back to ordinary-text handling. This targets Telegram's legacy
`Markdown` mode, not MarkdownV2.

For full control, use `format: HTML`:

```yaml
format: HTML
message: |
  <code>_site</code>
  <a href="https://example.com/_site">_site</a>
```

## Install

Prebuilt binaries for Linux, macOS, Windows and FreeBSD are on the
[releases page](https://github.com/alkk/forgejo-telegram/releases). With Go
installed:

```sh
go install github.com/alkk/forgejo-telegram@latest
```

## Development

```sh
make build   # bin/forgejo-telegram
make test
make lint
```

Tests that send real messages run only when `TELEGRAM_TOKEN` and `TELEGRAM_TO`
are set.

## License

[MIT](LICENSE)
