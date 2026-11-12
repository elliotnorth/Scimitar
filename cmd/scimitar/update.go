package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releaseRepo     = "elliotnorth/Scimitar"
	maxBinarySize   = 100 << 20
	downloadTimeout = 2 * time.Minute
)

type releases struct {
	apiBase      string
	downloadBase string
	client       *http.Client
}

func githubReleases() releases {
	return releases{
		apiBase:      "https://api.github.com",
		downloadBase: "https://github.com",
		client:       &http.Client{Timeout: downloadTimeout},
	}
}

func (r releases) latestTag(ctx context.Context) (string, error) {
	body, err := r.get(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", r.apiBase, releaseRepo), 1<<20)
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil || release.TagName == "" {
		return "", errors.New("could not read the latest release")
	}
	return release.TagName, nil
}

func (r releases) download(ctx context.Context, tag, asset string, limit int64) ([]byte, error) {
	return r.get(ctx, fmt.Sprintf("%s/%s/releases/download/%s/%s", r.downloadBase, releaseRepo, tag, asset), limit)
}

func (r releases) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scimitar-updater")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func assetName() string {
	return "scimitar-linux-" + runtime.GOARCH
}

func checksumFor(checksums []byte, asset string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(checksums)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum for %s in the release", asset)
}

func (r releases) fetchVerified(ctx context.Context, tag string) ([]byte, error) {
	checksums, err := r.download(ctx, tag, "checksums.txt", 1<<16)
	if err != nil {
		return nil, err
	}
	want, err := checksumFor(checksums, assetName())
	if err != nil {
		return nil, err
	}
	binary, err := r.download(ctx, tag, assetName(), maxBinarySize)
	if err != nil {
		return nil, err
	}
	got := sha256.Sum256(binary)
	if hex.EncodeToString(got[:]) != strings.ToLower(want) {
		return nil, errors.New("downloaded file does not match its checksum, nothing was changed")
	}
	return binary, nil
}

type updater struct {
	releases releases
	system   system
	out      io.Writer
}

func (u updater) update(ctx context.Context) error {
	latest, err := u.releases.latestTag(ctx)
	if err != nil {
		return err
	}
	if latest == version {
		fmt.Fprintf(u.out, "Scimitar %s is already the newest version.\n", version)
		return nil
	}
	fmt.Fprintf(u.out, "Updating Scimitar %s to %s...\n", version, latest)
	binary, err := u.releases.fetchVerified(ctx, latest)
	if err != nil {
		return err
	}
	if err := replaceFile(u.system.paths.binary, binary); err != nil {
		return err
	}
	if fileExists(u.system.paths.unit) {
		if err := u.system.run("systemctl", "restart", "scimitar"); err != nil {
			return err
		}
	}
	fmt.Fprintf(u.out, "Done. Scimitar %s is running.\n", latest)
	return nil
}

func replaceFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scimitar-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
