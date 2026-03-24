package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	githubAPIVersionHeader = "2022-11-28"
	updateCheckInterval    = 6 * time.Hour
	updateCheckStartupWait = 8 * time.Second
	updateHTTPTimeout      = 20 * time.Second
)

type githubRelease struct {
	TagName    string               `json:"tag_name"`
	Name       string               `json:"name"`
	Draft      bool                 `json:"draft"`
	Prerelease bool                 `json:"prerelease"`
	Assets     []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

type appUpdateState struct {
	Supported       bool
	Checking        bool
	Installing      bool
	UpdateAvailable bool
	CurrentVersion  string
	LatestVersion   string
	LastError       string
	LastCheckedAt   time.Time
}

type AppUpdater struct {
	mu             sync.RWMutex
	client         *http.Client
	repoOwner      string
	repoName       string
	currentVersion string
	supported      bool
	checking       bool
	installing     bool
	lastCheckedAt  time.Time
	lastError      string
	latest         *githubRelease
	onChange       func()
}

func NewAppUpdater(repoURL, currentVersion string) *AppUpdater {
	owner, repo, ok := parseGitHubRepository(repoURL)
	normalizedVersion := normalizeVersionString(currentVersion)
	supported := ok && normalizedVersion != "" && normalizedVersion != "unknown" && runtime.GOOS != "" && runtime.GOARCH != ""
	return &AppUpdater{
		client: &http.Client{
			Timeout: updateHTTPTimeout,
		},
		repoOwner:      owner,
		repoName:       repo,
		currentVersion: normalizedVersion,
		supported:      supported,
	}
}

func (u *AppUpdater) Start(stopCh <-chan struct{}, onChange func()) {
	u.mu.Lock()
	u.onChange = onChange
	supported := u.supported
	u.mu.Unlock()
	u.notifyChange()
	if !supported || stopCh == nil {
		return
	}
	go u.run(stopCh)
}

func (u *AppUpdater) run(stopCh <-chan struct{}) {
	startupTimer := time.NewTimer(updateCheckStartupWait)
	defer startupTimer.Stop()
	ticker := time.NewTicker(updateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-startupTimer.C:
			u.TriggerCheck()
			startupTimer = nil
		case <-ticker.C:
			u.TriggerCheck()
		}
	}
}

func (u *AppUpdater) State() appUpdateState {
	u.mu.RLock()
	defer u.mu.RUnlock()

	state := appUpdateState{
		Supported:      u.supported,
		Checking:       u.checking,
		Installing:     u.installing,
		CurrentVersion: u.currentVersion,
		LastError:      u.lastError,
		LastCheckedAt:  u.lastCheckedAt,
	}
	if u.latest != nil {
		state.LatestVersion = normalizeVersionString(u.latest.TagName)
		state.UpdateAvailable = isNewerVersion(state.CurrentVersion, state.LatestVersion)
	}
	return state
}

func (u *AppUpdater) TriggerCheck() bool {
	u.mu.Lock()
	if !u.supported || u.checking || u.installing {
		u.mu.Unlock()
		return false
	}
	u.checking = true
	u.lastError = ""
	u.mu.Unlock()
	u.notifyChange()

	go u.checkLatestRelease()
	return true
}

func (u *AppUpdater) checkLatestRelease() {
	ctx, cancel := context.WithTimeout(context.Background(), updateHTTPTimeout)
	defer cancel()

	release, err := u.fetchLatestRelease(ctx)
	u.mu.Lock()
	u.checking = false
	u.lastCheckedAt = time.Now()
	if err != nil {
		u.lastError = err.Error()
		u.mu.Unlock()
		logWarnModule("update", "check latest release failed: %v", err)
		u.notifyChange()
		return
	}

	oldLatest := ""
	if u.latest != nil {
		oldLatest = normalizeVersionString(u.latest.TagName)
	}
	u.latest = release
	u.lastError = ""
	state := u.stateLocked()
	u.mu.Unlock()

	if state.UpdateAvailable && state.LatestVersion != oldLatest {
		logInfoModule("update", "update available: current=%s latest=%s", state.CurrentVersion, state.LatestVersion)
	} else if state.LatestVersion != "" {
		logInfoModule("update", "update check complete: current=%s latest=%s", state.CurrentVersion, state.LatestVersion)
	}
	u.notifyChange()
}

func (u *AppUpdater) PrepareUpgrade(currentArgs []string) error {
	u.mu.Lock()
	if !u.supported {
		u.mu.Unlock()
		return fmt.Errorf("auto update unavailable for current build")
	}
	if u.installing {
		u.mu.Unlock()
		return fmt.Errorf("update already in progress")
	}
	u.installing = true
	u.lastError = ""
	u.mu.Unlock()
	u.notifyChange()

	err := u.prepareUpgradeLocked(currentArgs)

	u.mu.Lock()
	u.installing = false
	if err != nil {
		u.lastError = err.Error()
	}
	u.mu.Unlock()
	u.notifyChange()
	return err
}

func (u *AppUpdater) prepareUpgradeLocked(currentArgs []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateHTTPTimeout)
	defer cancel()

	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		return err
	}

	latestVersion := normalizeVersionString(release.TagName)
	if !isNewerVersion(u.currentVersion, latestVersion) {
		return fmt.Errorf("already on latest version %s", latestVersion)
	}

	asset, err := selectReleaseAsset(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	tempRoot, err := os.MkdirTemp("", "mrs-update-*")
	if err != nil {
		return fmt.Errorf("create update temp directory failed: %w", err)
	}

	archivePath := filepath.Join(tempRoot, asset.Name)
	if err := u.downloadAsset(ctx, asset, archivePath); err != nil {
		return err
	}

	extractDir := filepath.Join(tempRoot, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return fmt.Errorf("create extract directory failed: %w", err)
	}
	if err := extractReleaseArchive(archivePath, extractDir); err != nil {
		return err
	}

	packageRoot, err := resolvePackageRoot(extractDir)
	if err != nil {
		return err
	}

	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable failed: %w", err)
	}
	executablePath, _ = filepath.EvalSymlinks(executablePath)
	if executablePath == "" {
		executablePath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve current executable failed: %w", err)
		}
	}

	currentPID := os.Getpid()
	if err := scheduleUpdateAndRestart(packageRoot, executablePath, currentArgs, currentPID); err != nil {
		return err
	}

	u.mu.Lock()
	u.latest = release
	u.lastCheckedAt = time.Now()
	u.lastError = ""
	u.mu.Unlock()

	logInfoModule("update", "scheduled upgrade to %s from asset %s", latestVersion, asset.Name)
	return nil
}

func (u *AppUpdater) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", url.PathEscape(u.repoOwner), url.PathEscape(u.repoName))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build update request failed: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", githubAPIVersionHeader)
	request.Header.Set("User-Agent", "MetricsRenderSender-Updater")

	response, err := u.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request latest release failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("latest release request failed: %s", message)
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode latest release failed: %w", err)
	}
	if release.Draft {
		return nil, fmt.Errorf("latest release is draft")
	}
	return &release, nil
}

func (u *AppUpdater) downloadAsset(ctx context.Context, asset *githubReleaseAsset, destination string) error {
	if asset == nil {
		return fmt.Errorf("release asset is nil")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("build asset request failed: %w", err)
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "MetricsRenderSender-Updater")

	response, err := u.client.Do(request)
	if err != nil {
		return fmt.Errorf("download update asset failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("download update asset failed: %s", message)
	}

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create asset file failed: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, response.Body); err != nil {
		return fmt.Errorf("write asset file failed: %w", err)
	}
	return nil
}

func (u *AppUpdater) stateLocked() appUpdateState {
	state := appUpdateState{
		Supported:      u.supported,
		Checking:       u.checking,
		Installing:     u.installing,
		CurrentVersion: u.currentVersion,
		LastError:      u.lastError,
		LastCheckedAt:  u.lastCheckedAt,
	}
	if u.latest != nil {
		state.LatestVersion = normalizeVersionString(u.latest.TagName)
		state.UpdateAvailable = isNewerVersion(state.CurrentVersion, state.LatestVersion)
	}
	return state
}

func (u *AppUpdater) notifyChange() {
	u.mu.RLock()
	callback := u.onChange
	u.mu.RUnlock()
	if callback != nil {
		callback()
	}
}

func parseGitHubRepository(repositoryURL string) (string, string, bool) {
	text := strings.TrimSpace(repositoryURL)
	if text == "" {
		return "", "", false
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return "", "", false
	}
	if !strings.EqualFold(parsed.Host, "github.com") {
		return "", "", false
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 {
		return "", "", false
	}
	owner := strings.TrimSpace(segments[0])
	repo := strings.TrimSuffix(strings.TrimSpace(segments[1]), ".git")
	if owner == "" || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}

func normalizeVersionString(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	return value
}

func isNewerVersion(current, latest string) bool {
	return compareVersionStrings(normalizeVersionString(current), normalizeVersionString(latest)) < 0
}

func compareVersionStrings(left, right string) int {
	leftParts, leftPre := parseVersionParts(left)
	rightParts, rightPre := parseVersionParts(right)

	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}
	for i := 0; i < maxLen; i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			leftValue = leftParts[i]
		}
		if i < len(rightParts) {
			rightValue = rightParts[i]
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}

	switch {
	case leftPre == "" && rightPre != "":
		return 1
	case leftPre != "" && rightPre == "":
		return -1
	case leftPre < rightPre:
		return -1
	case leftPre > rightPre:
		return 1
	default:
		return 0
	}
}

func parseVersionParts(raw string) ([]int, string) {
	if raw == "" {
		return []int{0}, ""
	}
	base := raw
	pre := ""
	if idx := strings.IndexAny(raw, "-+"); idx >= 0 {
		base = raw[:idx]
		pre = raw[idx+1:]
	}
	segments := strings.Split(base, ".")
	parts := make([]int, 0, len(segments))
	for _, segment := range segments {
		value, err := strconv.Atoi(strings.TrimSpace(segment))
		if err != nil {
			break
		}
		parts = append(parts, value)
	}
	if len(parts) == 0 {
		parts = []int{0}
	}
	return parts, pre
}

func selectReleaseAsset(release *githubRelease, goos, goarch string) (*githubReleaseAsset, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}
	prefix, suffix, err := releaseAssetPattern(goos, goarch)
	if err != nil {
		return nil, err
	}
	for idx := range release.Assets {
		asset := &release.Assets[idx]
		name := strings.TrimSpace(asset.Name)
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			return asset, nil
		}
	}
	return nil, fmt.Errorf("no release asset found for %s/%s", goos, goarch)
}

func releaseAssetPattern(goos, goarch string) (string, string, error) {
	switch goos {
	case "linux":
		return fmt.Sprintf("metrics_render_sender-%s-%s-", goos, goarch), ".tar.gz", nil
	case "windows":
		return fmt.Sprintf("metrics_render_sender-%s-%s-", goos, goarch), ".zip", nil
	default:
		return "", "", fmt.Errorf("auto update unsupported on %s/%s", goos, goarch)
	}
}

func expectedPackageExecutableName(goos string) (string, error) {
	switch goos {
	case "linux":
		return "metrics_render_sender", nil
	case "windows":
		return "metrics_render_sender.exe", nil
	default:
		return "", fmt.Errorf("auto update unsupported on %s", goos)
	}
}

func extractReleaseArchive(archivePath, destination string) error {
	switch {
	case strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz"):
		return extractTarGZArchive(archivePath, destination)
	case strings.HasSuffix(strings.ToLower(archivePath), ".zip"):
		return extractZipArchive(archivePath, destination)
	default:
		return fmt.Errorf("unsupported archive format: %s", archivePath)
	}
}

func extractTarGZArchive(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open tar.gz archive failed: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("open gzip stream failed: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry failed: %w", err)
		}
		if err := writeArchiveEntry(destination, header.Name, header.FileInfo(), tarReader); err != nil {
			return err
		}
	}
}

func extractZipArchive(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip archive failed: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		entryReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("open zip entry failed: %w", err)
		}
		err = writeArchiveEntry(destination, file.Name, file.FileInfo(), entryReader)
		entryReader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeArchiveEntry(destination, name string, info fs.FileInfo, source io.Reader) error {
	if info == nil {
		return fmt.Errorf("archive entry info missing")
	}
	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == string(filepath.Separator) {
		return nil
	}
	if strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return fmt.Errorf("archive entry escapes destination: %s", name)
	}

	targetPath := filepath.Join(destination, cleanName)
	if info.IsDir() {
		if err := os.MkdirAll(targetPath, 0o755); err != nil {
			return fmt.Errorf("create directory %s failed: %w", targetPath, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %s failed: %w", targetPath, err)
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create file %s failed: %w", targetPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, source); err != nil {
		return fmt.Errorf("write file %s failed: %w", targetPath, err)
	}
	return nil
}

func resolvePackageRoot(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("read extracted directory failed: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractDir, entries[0].Name()), nil
	}
	return extractDir, nil
}

func copyFileWithMode(sourcePath, targetPath string, mode fs.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file failed: %w", err)
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target directory failed: %w", err)
	}
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("open target file failed: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("copy file failed: %w", err)
	}
	return nil
}
