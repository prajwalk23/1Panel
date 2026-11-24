package job

import "github.com/1Panel-dev/1Panel/core/app/service"

type upgradePlan struct{}

func NewUpgradePlanJob() *upgradePlan {
	return &upgradePlan{}
}

// Run checks the stored panel update plan (once / daily) and starts the upgrade when it is due.
func (j *upgradePlan) Run() {
	service.NewIUpgradeService().RunScheduledPlan()
}
