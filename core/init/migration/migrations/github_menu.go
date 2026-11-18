package migrations

import (
	"sort"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/init/migration/helper"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: adds the top-level GitHub menu right after Cron Jobs. Existing entries,
// their visibility and their order are kept; nothing happens if it already exists.
var AddGithubMenu = &gormigrate.Migration{
	ID: "20261118-fork-add-github-menu",
	Migrate: func(tx *gorm.DB) error {
		return helper.UpdateHideMenu(tx, addGithubMenu)
	},
}

func addGithubMenu(menus []dto.ShowMenu) []dto.ShowMenu {
	for _, menu := range menus {
		if menu.Label == helper.GithubMenu.Label {
			return menus
		}
	}
	item := helper.GithubMenu
	sorted := append([]dto.ShowMenu{}, menus...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Sort < sorted[j].Sort })
	for i := range sorted {
		if sorted[i].Label != "Cronjob-Menu" {
			continue
		}
		prev := sorted[i].Sort
		if i+1 < len(sorted) {
			next := sorted[i+1].Sort
			if next-prev >= 2 {
				item.Sort = prev + (next-prev)/2
				return append(menus, item)
			}
			// No room between the two: renumber, keeping the current order.
			result := make([]dto.ShowMenu, 0, len(sorted)+1)
			result = append(result, sorted[:i+1]...)
			result = append(result, item)
			result = append(result, sorted[i+1:]...)
			for j := range result {
				result[j].Sort = (j + 1) * 100
			}
			return result
		}
		item.Sort = prev + 50
		return append(menus, item)
	}
	return append(menus, item)
}
