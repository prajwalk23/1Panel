# Fork patches

This fork (`prajwalk23/1Panel`) adds a **GitHub** section to 1Panel. Almost all fork code lives in
new files. This page lists every **upstream file** the fork edits, so conflicts after an upstream
sync are quick to resolve. Keep it up to date whenever an upstream file is touched.

Base: upstream tag `v2.5.1` (806de63). Upstream merges come from stable tags only (see `fork/README.md`).

## Edited upstream files

| File | Change | On conflict |
|---|---|---|
| `agent/router/common.go` | `&GithubRouter{},` appended to `commonGroups()` | keep upstream list, re-append our line |
| `agent/init/migration/migrate.go` | `migrations.AddGithubTables,` and `migrations.AddGithubScripts,` appended to `agentDBMigrations()` | keep upstream entries, keep ours **last** (order only matters for new installs; IDs are unique) |
| `agent/init/business/business.go` | `go service.StartGithubWebhookListener()` in `Init()` | re-add the line |
| `agent/i18n/i18n.go` | `Init()` also loads `lang/github/*.yaml` via `loadGithubLangFiles()` | re-add the call |
| `core/router/common.go` | `&GithubRouter{},` appended to `commonGroups()` | re-append our line |
| `core/init/migration/migrate.go` | `migrations.AddGithubMenu,` appended | keep ours last |
| `core/init/migration/helper/menu.go` | `GitHub-Menu` entry in `LoadMenus()` and `MenuSort()` | re-add the entries |
| `core/app/service/upgrade.go` | fork block at the top of `SearchUpgrade` (answers from the fork releases) and `Upgrade` (hands over to the fork updater) | re-add both blocks |
| `core/i18n/i18n.go` | `Init()` also loads `lang/github/*.yaml` via `loadGithubLangFiles()` | re-add the call |
| `frontend/src/lang/index.ts` | messages pass through `withGithubMessages()` | re-wrap both places |

Every edit is marked with a `// fork:` comment so `git grep "fork:"` finds them all.

## Mirrored upstream logic (check after every sync)

`core/app/service/github_update.go` (`installForkPackage`) repeats the install steps of
`UpgradeService.Upgrade()` in `core/app/service/upgrade.go`: copy `1panel-core`/`1panel-agent`,
`1pctl` (+ `BASE_DIR`/`LANGUAGE` update), init scripts, `lang/`, `GeoIP.mmdb`, with rollback steps 1-4.
After an upstream sync, diff `Upgrade()` against `installForkPackage` and port any new step.

It also calls these package-private helpers from `upgrade.go`. A rename upstream breaks the build:
`loadArch`, `loadServiceInfo`, `checkUpgradeSpace`, `handleBackup`, `handleRollback`, `dropBackupCopies`.

## New files (no conflict risk)

- Agent: `agent/utils/github/`, `agent/app/model/github.go`, `agent/app/dto/github.go`,
  `agent/app/repo/github.go`, `agent/app/service/github*.go`, `agent/app/api/v2/github.go`,
  `agent/router/ro_github.go`, `agent/init/migration/migrations/github*.go`, `agent/i18n/lang/github/`,
  `agent/i18n/github.go`
- Core: `core/global/fork.go`, `core/utils/github/`, `core/app/service/github_update.go`,
  `core/app/dto/github.go`, `core/app/api/v2/github_update.go`, `core/router/ro_github.go`,
  `core/init/migration/migrations/github_menu.go`, `core/init/migration/helper/github_menu.go`,
  `core/i18n/lang/github/`, `core/i18n/github.go`
- Frontend: `frontend/src/routers/modules/github.ts`, `frontend/src/views/github/`,
  `frontend/src/api/modules/github.ts`, `frontend/src/api/interface/github.ts`, `frontend/src/lang/github/`
- Fork tooling: `.github/workflows/sync-upstream.yml`, `.github/workflows/fork-ci.yml`,
  `.github/workflows/release.yml`, `fork/`

## Upstream workflows

`sync2gitee.yml`, `issue-translator.yml` and `add-labels-for-pr.yml` are **deleted** in the fork.
The sync job always keeps the fork's `.github/workflows` as on `main` (it also resolves a
modify/delete conflict when upstream edits one of the deleted files).
