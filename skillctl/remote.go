package skillctl

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const defaultBaseURL = "https://api.skillctl.dev"

// Client is an API client for the skillctl remote registry.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a Client with default settings.
func NewClient() *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		HTTPClient: http.DefaultClient,
	}
}

// SkillInfo describes a skill returned by the remote API.
type SkillInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Downloads   int    `json:"downloads"`
	TarballURL  string `json:"tarball_url"`
}

// SearchResult holds the results of a skill search.
type SearchResult struct {
	Skills []SkillInfo `json:"skills"`
	Total  int         `json:"total"`
}

// Search queries the remote registry.
func (c *Client) Search(ctx context.Context, query string) (*SearchResult, error) {
	u := c.baseURL() + "/v1/skills?q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("skillctl: search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("skillctl: search: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("skillctl: search: decode: %w", err)
	}
	return &result, nil
}

// Info retrieves details for a single skill.
func (c *Client) Info(ctx context.Context, name string) (*SkillInfo, error) {
	u := c.baseURL() + "/v1/skills/" + url.PathEscape(name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("skillctl: info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skillctl: skill %q not found", name)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("skillctl: info: HTTP %d: %s", resp.StatusCode, string(body))
	}
	var info SkillInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("skillctl: info: decode: %w", err)
	}
	return &info, nil
}

// InstallOptions configures an Install operation.
type InstallOptions struct {
	Name         string // skill name (or name@version)
	SkillsDir    string // defaults to DefaultSkillsDir()
	RegistryPath string // defaults to DefaultRegistryPath()
	Force        bool   // overwrite existing
	Sync         bool   // run Sync after install
}

// InstallResult describes what Install did.
type InstallResult struct {
	Name       string
	Version    string
	InstallDir string
	Synced     bool
}

// Install downloads a skill tarball and extracts it to the skills directory.
func (c *Client) Install(ctx context.Context, opts InstallOptions) (*InstallResult, error) {
	name, version := parseName(opts.Name)

	skillsDir := opts.SkillsDir
	if skillsDir == "" {
		skillsDir = DefaultSkillsDir()
	}
	regPath := opts.RegistryPath
	if regPath == "" {
		regPath = DefaultRegistryPath()
	}

	// Fetch skill info to get tarball URL.
	lookupName := name
	if version != "" {
		lookupName = name + "@" + version
	}
	info, err := c.Info(ctx, lookupName)
	if err != nil {
		return nil, err
	}

	destDir := filepath.Join(skillsDir, name)
	if !opts.Force {
		if _, err := os.Stat(destDir); err == nil {
			return nil, fmt.Errorf("skillctl: %s already installed at %s (use --force to overwrite)", name, destDir)
		}
	}

	// Download and extract tarball.
	tarballURL := info.TarballURL
	if tarballURL == "" {
		tarballURL = c.baseURL() + "/v1/skills/" + url.PathEscape(name) + "/tarball"
	}

	if err := downloadAndExtract(ctx, c.httpClient(), tarballURL, destDir); err != nil {
		return nil, fmt.Errorf("skillctl: install %s: %w", name, err)
	}

	// Update registry.
	reg, err := LoadRegistry(regPath)
	if err != nil {
		return nil, err
	}
	reg.Skills[name] = SkillEntry{Enabled: true}
	if err := SaveRegistry(regPath, reg); err != nil {
		return nil, fmt.Errorf("skillctl: save registry: %w", err)
	}

	result := &InstallResult{
		Name:       name,
		Version:    info.Version,
		InstallDir: destDir,
	}

	if opts.Sync {
		if _, err := Sync(SyncOptions{
			RegistryPath: regPath,
			SkillsDir:    skillsDir,
		}); err != nil {
			return result, fmt.Errorf("skillctl: sync after install: %w", err)
		}
		result.Synced = true
	}

	return result, nil
}

func downloadAndExtract(ctx context.Context, client *http.Client, tarURL, destDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching tarball", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		// Strip the first path component (tarball root directory).
		name := hdr.Name
		if idx := strings.IndexByte(name, '/'); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)
		// Safety: prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// parseName splits "name@version" into (name, version).
func parseName(s string) (string, string) {
	if idx := strings.LastIndex(s, "@"); idx > 0 {
		return s[:idx], s[idx+1:]
	}
	return s, ""
}
