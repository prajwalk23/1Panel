package migrations

import (
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/init/migration/helper"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// fork: the AI and VM features were removed, so drop their entries from the saved menu.
// Without this the Settings > hide-menu tree keeps showing them as dead nodes. It runs
// last, after the upstream menu migrations that would otherwise (re-)add them.
var StripVMAndAIMenus = &gormigrate.Migration{
	ID: "20261124-fork-strip-vm-ai-menus",
	Migrate: func(tx *gorm.DB) error {
		return helper.UpdateHideMenu(tx, func(menus []dto.ShowMenu) []dto.ShowMenu {
			return stripMenuLabels(menus, map[string]bool{"AI-Menu": true, "VirtualMachine": true})
		})
	},
}

func stripMenuLabels(menus []dto.ShowMenu, labels map[string]bool) []dto.ShowMenu {
	result := make([]dto.ShowMenu, 0, len(menus))
	for _, menu := range menus {
		if labels[menu.Label] {
			continue
		}
		if len(menu.Children) > 0 {
			menu.Children = stripMenuLabels(menu.Children, labels)
		}
		result = append(result, menu)
	}
	return result
}
