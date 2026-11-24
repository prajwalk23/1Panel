package migrations

import (
	"testing"

	"github.com/1Panel-dev/1Panel/core/app/dto"
)

func TestStripVMAndAIMenus(t *testing.T) {
	menus := []dto.ShowMenu{
		{ID: "1", Label: "Home-Menu", Sort: 100},
		{ID: "3", Label: "AI-Menu", Sort: 300, Children: []dto.ShowMenu{{ID: "41", Label: "AIModel"}}},
		{ID: "123", Label: "VirtualMachine", Sort: 650},
		{ID: "11", Label: "Xpack-Menu", Sort: 1100, Children: []dto.ShowMenu{
			{ID: "118", Label: "XApp"},
			{ID: "123", Label: "VirtualMachine"},
			{ID: "117", Label: "XSetting"},
		}},
	}
	got := stripMenuLabels(menus, map[string]bool{"AI-Menu": true, "VirtualMachine": true})
	if len(got) != 2 || got[0].Label != "Home-Menu" || got[1].Label != "Xpack-Menu" {
		t.Fatalf("top level: %+v", got)
	}
	children := got[1].Children
	if len(children) != 2 || children[0].Label != "XApp" || children[1].Label != "XSetting" {
		t.Fatalf("nested VM entry not stripped: %+v", children)
	}
	if again := stripMenuLabels(got, map[string]bool{"AI-Menu": true, "VirtualMachine": true}); len(again) != 2 {
		t.Fatalf("not idempotent: %d", len(again))
	}
}
