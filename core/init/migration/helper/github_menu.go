package helper

import "github.com/1Panel-dev/1Panel/core/app/dto"

// fork: the top-level GitHub menu. Label must match the route name in
// frontend/src/routers/modules/github.ts.
var GithubMenu = dto.ShowMenu{ID: "201", Disabled: false, Title: "github.menu", IsShow: true, Label: "GitHub-Menu", Path: "/github", Sort: 950}
