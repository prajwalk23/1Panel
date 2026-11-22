# 1Panel fork with a GitHub section

This fork of [1Panel](https://github.com/1Panel-dev/1Panel) adds a **GitHub** item to the sidebar:

- Projects: connect a repo and branch to a folder on the server. The repository is checked with the chosen SSH key and its branches are listed
- Named SSH keys (ed25519) that you create in the form and add to the repository as deploy keys, or your own imported key
- Deploy scripts per project: several scripts, one default, each with its own run-as user
- Optional push-to-deploy webhook, set up when editing a project and linked to exactly one script
- "Pull now" (any script), deploy logs, cancel
- Fork Updates: update the panel from this fork's GitHub Releases

Licence: GPL-3.0, same as upstream. Source for every release is this repository.

## Branches

| Branch | Content | Written by |
|---|---|---|
| `main` | upstream stable tag + fork code. Default branch. | PRs and the sync job |
| `sync/upstream-<tag>` | upstream tag that did not merge cleanly | sync job (PR into `main`) |

`main` is based on upstream **stable tags only** (first base: `v2.5.1`). Unreleased upstream code
and its database migrations never reach `main`.

## Sync job (`.github/workflows/sync-upstream.yml`)

Runs daily and on demand:

1. Finds the newest upstream stable tag (`v2.X.Y`). Stops if `main` already contains it.
2. Merges the tag into `main`, keeping the fork's `.github/workflows`.
   - Clean merge: pushed to `main`. Nothing is built or deployed.
   - Conflict: pushes the tag as `sync/upstream-<tag>` and opens a PR. `main` is not touched.
3. Never force-pushes, resets or rebases.

Resolve a conflict PR locally: `git fetch origin && git switch main && git merge origin/sync/upstream-<tag>`,
fix the files listed in `FORK_PATCHES.md`, commit, push.

## Versions and releases

- The panel keeps reporting the **upstream** version (`v2.5.1`), so all upstream version logic is unchanged.
- The fork build is `<upstream version>.<revision>`, for example `v2.5.1.1`, `v2.5.1.2`, then `v2.5.2.1`
  after merging upstream `v2.5.2`. It is embedded in `1panel-core` and shown in GitHub > Fork Updates.
- A release is built by pushing a tag: `git tag v2.5.1.1 && git push origin v2.5.1.1`.
  `.github/workflows/release.yml` builds both binaries for amd64 and arm64, places them on top of the
  official package of the same upstream version, and publishes a GitHub Release with `checksums.txt`.

## Repository setup (once)

1. Push `main` and make it the default branch (Settings, General, Default branch).
2. Actions tab: enable workflows.
3. Settings, Actions, General: "Read and write permissions" and "Allow GitHub Actions to create and
   approve pull requests".
4. Create a fine-grained personal access token for this repository with **Contents**, **Workflows**
   and **Pull requests** set to read and write. Save it as the repository secret `SYNC_TOKEN`.
   The default Actions token cannot push upstream commits that touch workflow files.
5. Keep the repository public, so panels can download releases without a token.

## Installing on a server that runs official 1Panel

The official panel has no fork updater yet, so the first install uses the script attached to every
release. Test on a VM first.

1. Take a backup: 1Panel snapshot (System, Snapshot) and a copy of `<InstallDir>/1panel`.
2. The panel must run the same upstream version as the release base (for `v2.5.1.1`: official `v2.5.1`).
3. On the server, as root:

   ```bash
   curl -fsSLO https://github.com/prajwalk23/1Panel/releases/download/v2.5.1.1/fork-upgrade.sh
   bash fork-upgrade.sh v2.5.1.1
   ```

   The script downloads the package, checks its SHA-256, copies the current binaries to
   `<InstallDir>/1panel/fork-backup/<time>/`, swaps `1panel-core` and `1panel-agent` in place and
   restarts the panel. Data under `<InstallDir>/1panel` and Docker is not touched.
4. Roll back with `bash fork-upgrade.sh --rollback` (restores the last backup).

Later updates: GitHub > Fork Updates > Update. It uses the panel's own backup and rollback steps.

## Webhook setup

The webhook is optional and is configured when **editing** a project.

1. GitHub > Settings: type the webhook domain (for example `hooks.example.com`). It must not be the domain
   of a website in this panel (saving fails if it is), and it must point to this server.
2. Make that domain reach the local listener (`127.0.0.1:19090` by default) with a reverse proxy. The
   webhook dialog shows the nginx `location ^~ /github-hook/ { proxy_pass ...; }` block to copy.
3. Edit the project, turn the webhook on and pick the one script it runs. The webhook dialog shows the
   payload URL and the secret (the secret is shown once; regenerate it if lost).
4. On GitHub: repo Settings, Webhooks, Add webhook. Payload URL = the URL, content type
   `application/json`, secret = the secret, event = "Just the push event".
5. For SSH repositories, add the project's public key under repo Settings, Deploy keys.
