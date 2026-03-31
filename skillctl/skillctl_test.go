package skillctl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "registry.yaml")

	reg := &Registry{
		Version: 1,
		Skills: map[string]SkillEntry{
			"code-review": {Enabled: true, Clients: []string{"claude-code", "clawfirm"}},
			"disabled":    {Enabled: false},
		},
		Clients: map[string]ClientEntry{
			"clawfirm": {Enabled: true},
		},
	}

	if err := SaveRegistry(path, reg); err != nil {
		t.Fatalf("SaveRegistry: %v", err)
	}

	loaded, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if loaded.Version != 1 {
		t.Errorf("version: got %d, want 1", loaded.Version)
	}
	if se, ok := loaded.Skills["code-review"]; !ok || !se.Enabled {
		t.Errorf("code-review: got %+v", loaded.Skills["code-review"])
	}
	if se, ok := loaded.Skills["disabled"]; !ok || se.Enabled {
		t.Errorf("disabled: got %+v", loaded.Skills["disabled"])
	}
}

func TestLoadRegistryMissing(t *testing.T) {
	reg, err := LoadRegistry("/nonexistent/registry.yaml")
	if err != nil {
		t.Fatalf("LoadRegistry missing: %v", err)
	}
	if reg.Version != 1 {
		t.Errorf("version: got %d, want 1", reg.Version)
	}
	if len(reg.Skills) != 0 {
		t.Errorf("skills: got %d, want 0", len(reg.Skills))
	}
}

func TestSync(t *testing.T) {
	tmp := t.TempDir()
	skillsDir := filepath.Join(tmp, "skills")
	regPath := filepath.Join(tmp, "registry.yaml")
	clientDir := filepath.Join(tmp, "client-skills")

	// Create a skill.
	skillDir := filepath.Join(skillsDir, "my-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save registry with the skill enabled.
	reg := &Registry{
		Version: 1,
		Skills:  map[string]SkillEntry{"my-skill": {Enabled: true}},
		Clients: map[string]ClientEntry{"test-client": {Enabled: true}},
	}
	if err := SaveRegistry(regPath, reg); err != nil {
		t.Fatal(err)
	}

	// Override ClientDirs for testing.
	origDirs := ClientDirs
	ClientDirs = map[string]string{"test-client": clientDir}
	defer func() { ClientDirs = origDirs }()

	result, err := Sync(SyncOptions{
		RegistryPath: regPath,
		SkillsDir:    skillsDir,
	})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("Created: got %d, want 1", len(result.Created))
	}
	if result.Created[0].Skill != "my-skill" {
		t.Errorf("skill: got %q", result.Created[0].Skill)
	}

	// Verify symlink exists and points to the right place.
	linkPath := filepath.Join(clientDir, "my-skill")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != skillDir {
		t.Errorf("target: got %q, want %q", target, skillDir)
	}

	// Sync again — should skip (already exists).
	result2, err := Sync(SyncOptions{
		RegistryPath: regPath,
		SkillsDir:    skillsDir,
	})
	if err != nil {
		t.Fatalf("Sync 2: %v", err)
	}
	if len(result2.Created) != 0 {
		t.Errorf("Created on re-sync: got %d, want 0", len(result2.Created))
	}

	// Disable the skill and sync again — should remove symlink.
	reg.Skills["my-skill"] = SkillEntry{Enabled: false}
	if err := SaveRegistry(regPath, reg); err != nil {
		t.Fatal(err)
	}
	result3, err := Sync(SyncOptions{
		RegistryPath: regPath,
		SkillsDir:    skillsDir,
	})
	if err != nil {
		t.Fatalf("Sync 3: %v", err)
	}
	if len(result3.Removed) != 1 {
		t.Errorf("Removed: got %d, want 1", len(result3.Removed))
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Errorf("symlink still exists after removal")
	}
}

func TestSyncDryRun(t *testing.T) {
	tmp := t.TempDir()
	skillsDir := filepath.Join(tmp, "skills")
	regPath := filepath.Join(tmp, "registry.yaml")
	clientDir := filepath.Join(tmp, "client-skills")

	skillDir := filepath.Join(skillsDir, "my-skill")
	os.MkdirAll(skillDir, 0o755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0o644)

	reg := &Registry{
		Version: 1,
		Skills:  map[string]SkillEntry{"my-skill": {Enabled: true}},
		Clients: map[string]ClientEntry{},
	}
	SaveRegistry(regPath, reg)

	origDirs := ClientDirs
	ClientDirs = map[string]string{"test-client": clientDir}
	defer func() { ClientDirs = origDirs }()

	result, err := Sync(SyncOptions{
		RegistryPath: regPath,
		SkillsDir:    skillsDir,
		DryRun:       true,
	})
	if err != nil {
		t.Fatalf("Sync dry-run: %v", err)
	}
	if len(result.Created) != 1 {
		t.Fatalf("Created: got %d, want 1", len(result.Created))
	}
	// Verify no actual symlink was created.
	if _, err := os.Lstat(filepath.Join(clientDir, "my-skill")); !os.IsNotExist(err) {
		t.Error("symlink created during dry-run")
	}
}

func TestRemoteSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/skills" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query().Get("q")
		result := SearchResult{
			Total: 1,
			Skills: []SkillInfo{
				{Name: q + "-skill", Version: "1.0.0", Description: "A test skill"},
			},
		}
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	result, err := c.Search(context.Background(), "code-review")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total: got %d", result.Total)
	}
	if result.Skills[0].Name != "code-review-skill" {
		t.Errorf("name: got %q", result.Skills[0].Name)
	}
}

func TestRemoteInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/skills/my-skill" {
			json.NewEncoder(w).Encode(SkillInfo{
				Name:    "my-skill",
				Version: "2.0.0",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	info, err := c.Info(context.Background(), "my-skill")
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "my-skill" || info.Version != "2.0.0" {
		t.Errorf("info: got %+v", info)
	}

	// Not found.
	_, err = c.Info(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"code-review", "code-review", ""},
		{"code-review@1.0.0", "code-review", "1.0.0"},
		{"@scope/pkg@2.0", "@scope/pkg", "2.0"},
	}
	for _, tt := range tests {
		name, version := parseName(tt.input)
		if name != tt.wantName || version != tt.wantVersion {
			t.Errorf("parseName(%q) = (%q, %q), want (%q, %q)", tt.input, name, version, tt.wantName, tt.wantVersion)
		}
	}
}
