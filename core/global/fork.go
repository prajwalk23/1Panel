package global

// fork: set at release build time with
//   -ldflags "-X github.com/1Panel-dev/1Panel/core/global.ForkVersion=v2.5.1.1
//             -X github.com/1Panel-dev/1Panel/core/global.ForkRepo=owner/name"
// An empty ForkVersion means a plain upstream build (official updater stays active).
var (
	ForkVersion = ""
	ForkRepo    = "prajwalk23/1Panel"
)

// IsForkBuild reports whether this binary was built by the fork release workflow.
func IsForkBuild() bool {
	return ForkVersion != ""
}
