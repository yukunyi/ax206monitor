package main

import "testing"

func TestParseGitHubRepository(t *testing.T) {
	owner, repo, ok := parseGitHubRepository("https://github.com/yukunyi/metrics_render_sender")
	if !ok {
		t.Fatalf("expected repository to parse")
	}
	if owner != "yukunyi" || repo != "metrics_render_sender" {
		t.Fatalf("unexpected repository: owner=%q repo=%q", owner, repo)
	}
}

func TestCompareVersionStrings(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.0.0", right: "1.0.0", want: 0},
		{left: "1.0.0", right: "1.0.1", want: -1},
		{left: "1.2.0", right: "1.1.9", want: 1},
		{left: "v1.2.3", right: "1.2.10", want: -1},
		{left: "1.2.3-beta", right: "1.2.3", want: -1},
	}
	for _, tc := range cases {
		got := compareVersionStrings(normalizeVersionString(tc.left), normalizeVersionString(tc.right))
		if got != tc.want {
			t.Fatalf("compareVersionStrings(%q, %q) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestSelectReleaseAsset(t *testing.T) {
	release := &githubRelease{
		TagName: "v1.2.3",
		Assets: []githubReleaseAsset{
			{Name: "metrics_render_sender-linux-amd64-v1.2.3.tar.gz"},
			{Name: "metrics_render_sender-windows-amd64-v1.2.3.zip"},
		},
	}

	linuxAsset, err := selectReleaseAsset(release, "linux", "amd64")
	if err != nil {
		t.Fatalf("select linux asset failed: %v", err)
	}
	if linuxAsset.Name != "metrics_render_sender-linux-amd64-v1.2.3.tar.gz" {
		t.Fatalf("unexpected linux asset: %s", linuxAsset.Name)
	}

	windowsAsset, err := selectReleaseAsset(release, "windows", "amd64")
	if err != nil {
		t.Fatalf("select windows asset failed: %v", err)
	}
	if windowsAsset.Name != "metrics_render_sender-windows-amd64-v1.2.3.zip" {
		t.Fatalf("unexpected windows asset: %s", windowsAsset.Name)
	}
}
