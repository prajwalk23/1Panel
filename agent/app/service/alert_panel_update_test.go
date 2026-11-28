package service

import "testing"

func TestNewerForkRelease(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"newer revision", `{"tag_name":"v2.5.1.21"}`, "v2.5.1.21"},
		{"newer base", `{"tag_name":"v2.5.2.1"}`, "v2.5.2.1"},
		{"numeric compare", `{"tag_name":"v2.5.1.100"}`, "v2.5.1.100"},
		{"same", `{"tag_name":"v2.5.1.20"}`, ""},
		{"older", `{"tag_name":"v2.5.1.9"}`, ""},
		{"upstream style tag", `{"tag_name":"v2.5.2"}`, ""},
		{"draft", `{"tag_name":"v2.5.1.21","draft":true}`, ""},
		{"prerelease", `{"tag_name":"v2.5.1.21","prerelease":true}`, ""},
	}
	for _, c := range cases {
		got, err := newerForkRelease("v2.5.1.20", []byte(c.body))
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}
	if _, err := newerForkRelease("v2.5.1.20", []byte("<html>")); err == nil {
		t.Error("expected an error for a non JSON body")
	}
}
