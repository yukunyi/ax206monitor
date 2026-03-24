package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRealGitHubLatestReleaseDownloadAndStart(t *testing.T) {
	if os.Getenv("METRICS_RENDER_SENDER_RUN_REAL_UPDATE_TEST") != "1" {
		t.Skip("set METRICS_RENDER_SENDER_RUN_REAL_UPDATE_TEST=1 to run real GitHub update integration test")
	}

	updater := NewAppUpdater(RepositoryURL, "0.0.0")
	ctx, cancel := context.WithTimeout(context.Background(), updateAPIRequestTimeout)
	defer cancel()

	release, err := updater.fetchLatestRelease(ctx)
	if err != nil {
		t.Fatalf("fetchLatestRelease failed: %v", err)
	}
	asset, err := selectReleaseAsset(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("selectReleaseAsset failed: %v", err)
	}

	tempRoot := t.TempDir()
	archivePath := filepath.Join(tempRoot, asset.Name)
	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), updateAssetTimeout)
	defer downloadCancel()
	if err := updater.downloadAsset(downloadCtx, asset, archivePath); err != nil {
		t.Fatalf("downloadAsset failed: %v", err)
	}

	extractDir := filepath.Join(tempRoot, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("create extract dir failed: %v", err)
	}
	if err := extractReleaseArchive(archivePath, extractDir); err != nil {
		t.Fatalf("extractReleaseArchive failed: %v", err)
	}

	packageRoot, err := resolvePackageRoot(extractDir)
	if err != nil {
		t.Fatalf("resolvePackageRoot failed: %v", err)
	}
	executableName, err := expectedPackageExecutableName(runtime.GOOS)
	if err != nil {
		t.Fatalf("expectedPackageExecutableName failed: %v", err)
	}

	executablePath := filepath.Join(packageRoot, executableName)
	info, err := os.Stat(executablePath)
	if err != nil {
		t.Fatalf("downloaded executable missing: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("downloaded executable path is directory: %s", executablePath)
	}

	command := exec.Command(executablePath, "--list-monitors")
	command.Dir = packageRoot
	command.Env = append(os.Environ(), "METRICS_RENDER_SENDER_WEB=0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("downloaded executable failed to start: %v\n%s", err, string(output))
	}
	if len(output) == 0 {
		t.Fatalf("downloaded executable produced no output")
	}
}
