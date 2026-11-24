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
| `agent/init/business/business.go` | `go service.InitGithub()` in `Init()` | re-add the line |
| `agent/i18n/i18n.go` | `Init()` also loads `lang/github/*.yaml` via `loadGithubLangFiles()` | re-add the call |
| `core/router/common.go` | `&GithubRouter{},` appended to `commonGroups()` | re-append our line |
| `core/init/migration/migrate.go` | `migrations.AddGithubMenu,` appended | keep ours last |
| `core/init/migration/helper/menu.go` | `GitHub-Menu` entry in `LoadMenus()` and `MenuSort()` | re-add the entries |
| `core/app/dto/setting.go`, `core/app/service/setting.go`, `frontend/src/api/interface/setting.ts`, `frontend/src/components/system-upgrade/index.vue` | `ForkVersion` field in `SettingBaseInfo`, filled from `global.ForkVersion`, and the footer shows it instead of `systemVersion` | re-add the field and the one-line `forkVersion \|\| systemVersion` |
| `core/app/service/upgrade.go` | fork block at the top of `SearchUpgrade` (answers from the fork releases) and `Upgrade` (hands over to the fork updater) | re-add both blocks |
| `core/i18n/i18n.go` | `Init()` also loads `lang/github/*.yaml` via `loadGithubLangFiles()` | re-add the call |
| `frontend/src/lang/index.ts` | messages pass through `withGithubMessages()` | re-wrap both places |
| `agent/init/migration/migrate.go` | also `migrations.AddRuntimeGithubFields,`, `AddWebsiteGithubFields,` and `AddGithubManager,` appended | keep ours **last** |
| `agent/app/model/runtime.go` | `GitMode`/`GitCloneURL`/`GitBranch`/`GitKeyID`/`GitUpdateStrategy` columns on `Runtime` | re-add the 5 fields |
| `agent/app/dto/request/runtime.go`, `agent/app/dto/response/runtime.go` | git source fields on `RuntimeCreate`/`RuntimeUpdate`/`RuntimeDTO` | re-add the fields |
| `agent/app/service/runtime.go` | `validateRuntimeGit`/`applyRuntimeGitFromCreate` on create, clone + project link in `executeRuntimeCreateTask`, `githubDeleteLinked` in runtime delete, code-dir existence check relaxed for git | re-add the calls |
| `frontend/src/views/website/runtime/{python,node,java,go,dotnet}/operate/index.vue` | `<GitSource>` (create mode only) + `gitSource`/`gitEnabled` state, mapped into the create request | re-add the component and the apply block in `submit` |
| `frontend/src/views/website/runtime/{python,node,java,go,dotnet}/index.vue` | `GitHub` tag + "Git Manager" button on rows with `gitMode === 'git'`, `<GitManagerDrawer>` | re-add the button, tag click and drawer |
| `frontend/src/lang/modules/en.ts`, `zh.ts` | `runtime.git*` keys | re-add the keys |
| `agent/app/model/website.go` | git source columns on `Website`, plus transient `GitAuthMode`/`GitTokenUser`/`GitToken` (`gorm:"-"`) | re-add the fields |
| `agent/app/dto/request/website.go` | git fields incl. `GitAuthMode`/`GitTokenUser`/`GitToken` on `WebsiteCreate` | re-add the fields |
| `agent/app/service/website.go` | validate + apply the git source for PHP runtime and static sites, `githubLinkWebsite` after create, `githubDeleteLinked` in delete, template copy skipped for git sites | re-add the calls |
| `agent/app/service/website_utils.go` | clone the repo instead of the default index file for git PHP sites | re-add the branch |
| `frontend/src/api/interface/website.ts` | git fields incl. auth on `WebSiteCreateReq`, `gitMode` on `WebsiteDTO` | re-add the fields |
| `frontend/src/views/website/website/create/index.vue` | `<GitSource>` for PHP runtime and static sites, request mapping in `submit` | re-add the component and mapping |
| `frontend/src/lang/modules/en.ts` | `enableHTTPS` label "Enable HTTPS" | re-apply the label |
| `agent/init/router/router.go` | `registerGithubHook(Router)` | re-add the line |
| `core/init/router/router.go` | `githubHookSourcePolicy()` passed to `WhiteAllow`, `registerGithubHook(Router)` after `registerMCPTransport` | re-add both |
| `core/constant/common.go` | `/github*` paths in `WebUrlMap` | re-add the entries |
| `frontend/src/views/website/website/config/basic/index.vue` | `Git` tab (`name="17"`) when `website.gitMode === 'git'` | re-add the pane and import |
| `frontend/src/api/interface/runtime.ts` | `gitMode`/`gitCloneURL` on `Runtime`, `gitAuthMode`/`gitTokenUser`/`gitToken` on `RuntimeCreate` | re-add the fields |

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
  `agent/i18n/github.go`, `agent/app/service/runtime_github.go`,
  `agent/init/migration/migrations/runtime_github.go`,
  `agent/init/migration/migrations/website_github.go`, `agent/init/migration/migrations/github_aapanel.go`,
  `agent/app/service/github_{address,hook,history,link}.go`, `agent/init/router/github_hook.go`, `agent/utils/github/hook.go`
- Frontend: `frontend/src/views/github/components/` (`git-source` create form, `source-fields`, `git-manager`), `frontend/src/views/github/composables/`, `frontend/src/views/website/website/config/basic/git/`
- Core: `core/init/router/github_hook.go` (public `/hook` webhook), `core/global/fork.go`, `core/utils/github/`, `core/app/service/github_update.go`,
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
