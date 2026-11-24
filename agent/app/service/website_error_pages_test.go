package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"unicode"
)

func TestForkErrorPageRender(t *testing.T) {
	for _, code := range errorPageCodes {
		spec, ok := defaultHtmlSpecFor("/root", code)
		if !ok {
			t.Fatalf("no default for error page %s", code)
		}
		if strings.Contains(spec.content, "{{") {
			t.Errorf("error page %s still has an unreplaced placeholder", code)
		}
		def := errorPageDefs[code]
		if !strings.Contains(spec.content, def.Code) || !strings.Contains(spec.content, def.Title) || !strings.Contains(spec.content, def.Message) {
			t.Errorf("error page %s is missing its code, title or message", code)
		}
		if spec.path != "/root/error/"+code+".html" {
			t.Errorf("error page %s path = %s", code, spec.path)
		}
		if !isErrorPageType(code) {
			t.Errorf("%s should be an error page type", code)
		}
	}
}

func TestForkErrorPageDefaultsAreEnglish(t *testing.T) {
	for _, resourceType := range []string{"404", "php", "index", "domain404", "stop", "400", "401", "403", "500", "502", "503", "504"} {
		spec, ok := defaultHtmlSpecFor("/root", resourceType)
		if !ok || spec.content == "" {
			t.Fatalf("missing default for %s", resourceType)
		}
		for _, r := range spec.content {
			if unicode.Is(unicode.Han, r) {
				t.Errorf("default for %s contains a Han character %q", resourceType, r)
				break
			}
		}
	}
}

func TestForkDefaultHtmlSpecPaths(t *testing.T) {
	cases := map[string]string{
		"404":       "/root/default/404.html",
		"php":       "/root/default/index.php",
		"index":     "/root/default/index.html",
		"domain404": "/root/404.html",
		"stop":      "/root/stop/index.html",
	}
	for resourceType, want := range cases {
		spec, ok := defaultHtmlSpecFor("/root", resourceType)
		if !ok || spec.path != want {
			t.Errorf("%s: path = %q ok = %v, want %q", resourceType, spec.path, ok, want)
		}
		if isErrorPageType(resourceType) {
			t.Errorf("%s must not be treated as an error page type", resourceType)
		}
	}
	if _, ok := defaultHtmlSpecFor("/root", "nope"); ok {
		t.Error("unknown type should have no spec")
	}
}

func TestForkErrorPagesSnippet(t *testing.T) {
	snippet := errorPagesSnippetContent()
	for _, code := range errorPageCodes {
		if !strings.Contains(snippet, "error_page "+code+" "+errorPagesURLPrefix+code+".html;") {
			t.Errorf("snippet is missing error_page %s", code)
		}
	}
	if !strings.Contains(snippet, "location ^~ "+errorPagesURLPrefix+" {") || !strings.Contains(snippet, "internal;") {
		t.Error("snippet is missing the internal location")
	}
	if !strings.Contains(snippet, "alias "+errorPagesContainerRoot+"/"+errorPagesDir+"/;") {
		t.Error("snippet alias does not point at the error page directory")
	}
	if got := errorPagesInclude(); got != errorPagesContainerRoot+"/"+errorPagesDir+"/"+errorPagesSnippet {
		t.Errorf("include path = %s", got)
	}
}

func TestForkStaleStockPage(t *testing.T) {
	page := "<html>\r\n<body>stock</body>\r\n</html>\r\n"
	sum := sha256.Sum256([]byte("<html>\n<body>stock</body>\n</html>"))
	hash := hex.EncodeToString(sum[:])
	if !isStaleStockPage(page, hash) {
		t.Error("a page equal to the stock copy should match, whatever its line endings")
	}
	if isStaleStockPage(page+"<!-- edited -->", hash) {
		t.Error("an edited page must not match")
	}
	if isStaleStockPage(page, "") {
		t.Error("an empty hash must never match")
	}
}

func TestForkErrorPagesSwitchState(t *testing.T) {
	if !errorPagesEnabledIn("") {
		t.Error("a missing snippet counts as on")
	}
	if !errorPagesEnabledIn(errorPagesSnippetContent()) {
		t.Error("the full snippet means on")
	}
	if errorPagesEnabledIn(errorPagesDisabledSnippet) {
		t.Error("the disabled snippet means off")
	}
	if errorPagesEnabledIn("\r\n" + errorPagesDisabledSnippet + "\r\n") {
		t.Error("surrounding whitespace and line endings must not matter")
	}
	if strings.Contains(errorPagesDisabledSnippet, "error_page ") {
		t.Error("the disabled snippet must not define an error_page directive")
	}
}
