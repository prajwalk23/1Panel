package migrations

import (
	"testing"

	"github.com/1Panel-dev/1Panel/core/app/dto"
)

func TestAddGithubMenu(t *testing.T) {
	menus := []dto.ShowMenu{
		{ID: "9", Label: "Toolbox-Menu", Sort: 1000, IsShow: false},
		{ID: "10", Label: "Cronjob-Menu", Sort: 900, IsShow: true},
		{ID: "1", Label: "Home-Menu", Sort: 100},
	}
	got := addGithubMenu(menus)
	if len(got) != 4 {
		t.Fatalf("len %d", len(got))
	}
	var github dto.ShowMenu
	for _, m := range got {
		if m.Label == "GitHub-Menu" {
			github = m
		}
		if m.Label == "Toolbox-Menu" && (m.Sort != 1000 || m.IsShow) {
			t.Fatalf("existing entry changed: %+v", m)
		}
	}
	if github.Sort != 950 || !github.IsShow || github.Path != "/github" {
		t.Fatalf("github entry: %+v", github)
	}
	if again := addGithubMenu(got); len(again) != 4 {
		t.Fatalf("not idempotent: %d", len(again))
	}

	tight := []dto.ShowMenu{{ID: "10", Label: "Cronjob-Menu", Sort: 900}, {ID: "9", Label: "Toolbox-Menu", Sort: 901}}
	got = addGithubMenu(tight)
	if len(got) != 3 || got[0].Label != "Cronjob-Menu" || got[1].Label != "GitHub-Menu" || got[2].Label != "Toolbox-Menu" {
		t.Fatalf("tight order: %+v", got)
	}
	if got[0].Sort >= got[1].Sort || got[1].Sort >= got[2].Sort {
		t.Fatalf("tight sorts: %+v", got)
	}

	last := addGithubMenu([]dto.ShowMenu{{ID: "10", Label: "Cronjob-Menu", Sort: 900}})
	if len(last) != 2 || last[1].Sort != 950 {
		t.Fatalf("cronjob last: %+v", last)
	}
	none := addGithubMenu([]dto.ShowMenu{{ID: "1", Label: "Home-Menu", Sort: 100}})
	if len(none) != 2 {
		t.Fatalf("no cronjob: %+v", none)
	}
}
