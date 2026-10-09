package tools_upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Options struct {
	Repo           string
	CurrentVersion string
	Force          bool
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func Upgrade(opts Options) error {
	if opts.Repo == "" {
		opts.Repo = "yvvw/my-packages"
	}

	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	switch targetArch {
	case "amd64", "arm64":
		// supported
	default:
		return fmt.Errorf("unsupported %s", targetArch)
	}

	fmt.Println("Checking releases...")
	releases, err := fetchReleases(opts.Repo)
	if err != nil {
		return err
	}

	asset, latestVersion, err := findMatchingAsset(releases, targetOS, targetArch)
	if err != nil {
		return err
	}

	localVersion := opts.CurrentVersion
	if localVersion == "" {
		localVersion = "0"
	}

	if !opts.Force && localVersion != "0" && localVersion == latestVersion {
		fmt.Printf("sing-box is already up to date (%s), skip.\n", localVersion)
		return nil
	}

	targetPath, err := resolveTargetPath()
	if err != nil {
		return fmt.Errorf("resolve target path failed: %w", err)
	}

	fmt.Printf("Upgrading sing-box (%s -> %s)...\n", localVersion, latestVersion)
	fmt.Printf("Target binary: %s\n", targetPath)
	fmt.Printf("Downloading %s\n", asset.BrowserDownloadURL)

	if err := downloadAndExtract(asset.BrowserDownloadURL, asset.Name, targetPath); err != nil {
		return err
	}

	fmt.Printf("Successfully upgraded sing-box to %s\n", latestVersion)

	if _, err := os.Stat("/etc/init.d/sing-box"); err == nil {
		fmt.Println("Note: If sing-box is running as a daemon, run '/etc/init.d/sing-box restart' to apply changes.")
	} else if _, err := exec.LookPath("systemctl"); err == nil {
		fmt.Println("Note: If sing-box is running as a daemon, run 'systemctl restart sing-box' to apply changes.")
	}

	return nil
}

func fetchReleases(repo string) ([]githubRelease, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("fetch releases failed: HTTP %s (%s)", resp.Status, strings.TrimSpace(string(body)))
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode releases JSON failed: %w", err)
	}
	return releases, nil
}

func findMatchingAsset(releases []githubRelease, targetOS, targetArch string) (*githubAsset, string, error) {
	osPattern := targetOS
	archPattern := targetArch

	var targetRelease *githubRelease
	for i := range releases {
		if releases[i].TagName == "sing-box" {
			targetRelease = &releases[i]
			break
		}
	}
	if targetRelease == nil && len(releases) > 0 {
		targetRelease = &releases[0]
	}
	if targetRelease == nil {
		return nil, "", errors.New("no releases found")
	}

	matchSubstr := fmt.Sprintf("-%s-%s.", osPattern, archPattern)
	for i := range targetRelease.Assets {
		asset := &targetRelease.Assets[i]
		if strings.HasPrefix(asset.Name, "sing-box-") && strings.Contains(asset.Name, matchSubstr) {
			version := extractVersion(asset.Name, osPattern, archPattern)
			return asset, version, nil
		}
	}

	return nil, "", fmt.Errorf("no matching asset found for %s-%s in release %s", osPattern, archPattern, targetRelease.TagName)
}

func extractVersion(assetName, targetOS, targetArch string) string {
	prefix := "sing-box-"
	suffix := fmt.Sprintf("-%s-%s", targetOS, targetArch)

	idxStart := strings.Index(assetName, prefix)
	if idxStart == -1 {
		return ""
	}
	after := assetName[idxStart+len(prefix):]
	idxEnd := strings.LastIndex(after, suffix)
	if idxEnd == -1 {
		return ""
	}
	return after[:idxEnd]
}

func resolveTargetPath() (string, error) {
	execPath, err := os.Executable()
	if err == nil && execPath != "" {
		if evalPath, err := filepath.EvalSymlinks(execPath); err == nil {
			return evalPath, nil
		}
		return execPath, nil
	}
	if runtime.GOOS == "windows" {
		return filepath.Abs("sing-box.exe")
	}
	return "/usr/bin/sing-box", nil
}

func downloadAndExtract(downloadURL, assetName, destPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %s", resp.Status)
	}

	targetDir := filepath.Dir(destPath)
	tmpFile, err := os.CreateTemp(targetDir, ".sing-box-upgrade-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file in %s failed: %w", targetDir, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	var binaryExtracted bool

	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("read gzip archive failed: %w", err)
		}
		defer gzr.Close()

		tr := tar.NewReader(gzr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				_ = tmpFile.Close()
				return fmt.Errorf("read tar archive failed: %w", err)
			}
			if hdr.Typeflag != tar.TypeReg {
				continue
			}
			base := filepath.Base(hdr.Name)
			if base == "sing-box" || base == "sing-box.exe" {
				if _, err := io.Copy(tmpFile, tr); err != nil {
					_ = tmpFile.Close()
					return fmt.Errorf("extract binary failed: %w", err)
				}
				binaryExtracted = true
				break
			}
		}
	} else if strings.HasSuffix(assetName, ".zip") {
		tmpZip, err := os.CreateTemp("", "sing-box-zip-*.zip")
		if err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("create temp zip failed: %w", err)
		}
		defer func() {
			_ = os.Remove(tmpZip.Name())
		}()

		_, err = io.Copy(tmpZip, resp.Body)
		_ = tmpZip.Close()
		if err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("download zip content failed: %w", err)
		}

		zr, err := zip.OpenReader(tmpZip.Name())
		if err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("open zip failed: %w", err)
		}
		defer zr.Close()

		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			base := filepath.Base(f.Name)
			if base == "sing-box" || base == "sing-box.exe" {
				rc, err := f.Open()
				if err != nil {
					_ = tmpFile.Close()
					return fmt.Errorf("open zip entry failed: %w", err)
				}
				_, err = io.Copy(tmpFile, rc)
				_ = rc.Close()
				if err != nil {
					_ = tmpFile.Close()
					return fmt.Errorf("extract binary from zip failed: %w", err)
				}
				binaryExtracted = true
				break
			}
		}
	} else {
		_ = tmpFile.Close()
		return fmt.Errorf("unsupported archive format: %s", assetName)
	}

	if !binaryExtracted {
		_ = tmpFile.Close()
		return errors.New("binary sing-box not found in downloaded archive")
	}

	if err := tmpFile.Chmod(0755); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temporary binary failed: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary file failed: %w", err)
	}

	return replaceBinary(tmpPath, destPath)
}

func replaceBinary(tmpPath, targetPath string) error {
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(targetPath, oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("rename existing binary failed: %w", err)
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Rename(oldPath, targetPath)
			return fmt.Errorf("replace binary failed: %w", err)
		}
		_ = os.Remove(oldPath)
		return nil
	}

	// Linux / Unix daemon replacement handling:
	// A running process/daemon holds an active reference to the executable's inode.
	// 1. Direct write/truncate to targetPath fails with ETXTBSY ("text file busy").
	// 2. os.Rename(tmpPath, targetPath) does NOT open targetPath for writing;
	//    it atomically replaces the directory entry (dentry) directly in the VFS.
	//    The running daemon continues execution from its unlinked inode until restarted.
	err := os.Rename(tmpPath, targetPath)
	if err == nil {
		return nil
	}

	// If atomic rename fails (e.g. cross-filesystem boundary or specific overlayfs limitations):
	// Explicitly unlink targetPath first. On Linux, unlinking a running binary always succeeds.
	_ = os.Remove(targetPath)

	err = os.Rename(tmpPath, targetPath)
	if err == nil {
		return nil
	}

	// Fallback to copy into newly created inode (since old was unlinked, no ETXTBSY)
	return copyFile(tmpPath, targetPath, 0755)
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
