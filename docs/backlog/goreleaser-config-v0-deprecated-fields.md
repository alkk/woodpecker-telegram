---
worth: yes
where: .goreleaser.yaml
added: 2026-11-27
---
# GoReleaser config is version 0 and uses deprecated fields

GoReleaser 2.20.3 `goreleaser check` exits nonzero on `.goreleaser.yaml`. It says "only version: 2 configuration files are supported, yours is version: 0" and flags `snapshot.name_template` and `archives.format` as deprecated. `.github/workflows/goreleaser.yml` runs `version: latest`, so a future GoReleaser release could turn these warnings into a failed release. A snapshot still builds today, with warnings (checked during the Woodpecker-support plan review).

Fix: add `version: 2`, rename `snapshot.name_template` to `snapshot.version_template`, and change `archives.format` to `archives.formats`. Re-run `goreleaser check` and a `goreleaser build --snapshot --single-target --clean`. Inherited from upstream, not introduced by the fork.
