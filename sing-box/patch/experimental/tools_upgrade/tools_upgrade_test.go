package tools_upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractVersion(t *testing.T) {
	tests := []struct {
		assetName   string
		os          string
		arch        string
		expectedVer string
	}{
		{
			assetName:   "sing-box-1.15.0-alpha.11-SNAPSHOT-6afeff4-linux-amd64.tar.gz",
			os:          "linux",
			arch:        "amd64",
			expectedVer: "1.15.0-alpha.11-SNAPSHOT-6afeff4",
		},
		{
			assetName:   "sing-box-1.14.0-linux-arm64.tar.gz",
			os:          "linux",
			arch:        "arm64",
			expectedVer: "1.14.0",
		},
		{
			assetName:   "sing-box-1.14.1-windows-amd64.zip",
			os:          "windows",
			arch:        "amd64",
			expectedVer: "1.14.1",
		},
	}

	for _, tc := range tests {
		actual := extractVersion(tc.assetName, tc.os, tc.arch)
		if actual != tc.expectedVer {
			t.Errorf("extractVersion(%q, %q, %q) = %q, expected %q", tc.assetName, tc.os, tc.arch, actual, tc.expectedVer)
		}
	}
}

func TestFindMatchingAsset(t *testing.T) {
	releases := []githubRelease{
		{
			TagName: "bemfa",
			Assets: []githubAsset{
				{
					Name:               "bemfa-sing-box-1.14.0-testing-SNAPSHOT-1975772-linux-arm64.tar.gz",
					BrowserDownloadURL: "http://example.com/bemfa",
				},
			},
		},
		{
			TagName: "sing-box",
			Assets: []githubAsset{
				{
					Name:               "sing-box-1.15.0-alpha.11-SNAPSHOT-6afeff4-linux-amd64.tar.gz",
					BrowserDownloadURL: "http://example.com/amd64",
				},
				{
					Name:               "sing-box-1.15.0-alpha.11-SNAPSHOT-6afeff4-linux-arm64.tar.gz",
					BrowserDownloadURL: "http://example.com/arm64",
				},
			},
		},
	}

	asset, ver, err := findMatchingAsset(releases, "linux", "arm64")
	if err != nil {
		t.Fatalf("findMatchingAsset error: %v", err)
	}
	if asset.Name != "sing-box-1.15.0-alpha.11-SNAPSHOT-6afeff4-linux-arm64.tar.gz" {
		t.Errorf("unexpected asset name: %s", asset.Name)
	}
	if ver != "1.15.0-alpha.11-SNAPSHOT-6afeff4" {
		t.Errorf("unexpected version: %s", ver)
	}
}

func TestReplaceBinary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "upgrade-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	targetFile := filepath.Join(tmpDir, "sing-box")
	if err := os.WriteFile(targetFile, []byte("old-binary-content"), 0755); err != nil {
		t.Fatal(err)
	}

	newTmpFile := filepath.Join(tmpDir, "sing-box.tmp")
	if err := os.WriteFile(newTmpFile, []byte("new-binary-content"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(newTmpFile, targetFile); err != nil {
		t.Fatalf("replaceBinary failed with open handle: %v", err)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new-binary-content" {
		t.Fatalf("content mismatch, got %s", string(content))
	}
}
