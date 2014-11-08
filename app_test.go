package scroll_test

import (
	"bytes"
	"flag"
	"os/exec"
	"testing"
	"time"

	"github.com/mailgun/scroll/testutils"
)

var ldtest = flag.Bool("ldtest", false, "Run special test that requires -ldflags to be set at build time.")

func TestBuildInfo(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if !*ldtest {
		t.Skip("This test requires a binary with -ldflags set at build time.")
	}
	app := testutils.NewTestApp()
	info := app.Get(t, app.GetURL()+"/build_info")
	commit, description, link := getBuildInfo(t)
	wanted := []struct {
		field, want string
	}{
		{"commit", commit},
		{"description", description},
		{"github link", link},
	}
	for _, w := range wanted {
		if info[w.field] != w.want {
			t.Errorf("GET /build_info: got %#v for %s, want %q", info[w.field], w.field, w.want)
		}
	}
	buildTimeS, ok := info["build time"].(string)
	if !ok {
		t.Fatalf("Expected string 'build time' field, got %T", info["build time"])
	}
	buildTime, err := time.Parse(time.UnixDate, buildTimeS)
	if err != nil {
		t.Fatalf("Could not parse build time %q", buildTimeS)
	}
	now := time.Now()
	if now.Before(buildTime) || now.Sub(buildTime) > time.Minute {
		t.Errorf("GET /build_info: Got a build time of %v. This looks wrong because the current time is %v.", buildTime, now)
	}
}

func getBuildInfo(t *testing.T) (commit, description, link string) {
	out, err := exec.Command("git", "log", "--max-count=1", "--oneline").CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %v\n\nOutput from git log:\n%s", err, out)
	}
	parts := bytes.SplitN(out, []byte(" "), 2)
	commit = string(parts[0])
	description = string(bytes.TrimSpace(parts[1]))
	link = "https://github.com/mailgun/scroll/commit/" + commit
	return
}
