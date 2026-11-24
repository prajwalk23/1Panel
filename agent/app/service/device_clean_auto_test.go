package service

import "testing"

func TestForkCleanMode(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"   ", ""},
		{"not json", ""},
		{"{}", ""},
		{`{"forkClean":"auto"}`, forkCleanAuto},
		{`{"forkClean":"upgradeRollback"}`, forkCleanUpgradeRollback},
		{`{"forkClean":"something-else"}`, ""},
		{`{"retainDays":7,"forkClean":"auto"}`, forkCleanAuto},
	}
	for _, c := range cases {
		if got := forkCleanMode(c.in); got != c.want {
			t.Errorf("forkCleanMode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSameVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v2.5.1.3", "2.5.1.3", true},
		{"2.5.1.3", "v2.5.1.3", true},
		{" v2.5.1.3 ", "v2.5.1.3", true},
		{"v2.5.1.3", "v2.5.1.2", false},
		{"", "", false},
		{"v2.5.1", "", false},
	}
	for _, c := range cases {
		if got := sameVersion(c.a, c.b); got != c.want {
			t.Errorf("sameVersion(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
