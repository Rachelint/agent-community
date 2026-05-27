package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProjectContextCreatesSkillLinksAndExclude(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoDir, ".git", "info"), 0o755); err != nil {
		t.Fatalf("mkdir .git/info: %v", err)
	}
	agentsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentsDir, "shared-skills", "issue-plan"), 0o755); err != nil {
		t.Fatalf("mkdir shared-skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "shared-skills", "issue-plan", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write shared skill: %v", err)
	}
	dirs := DataDirs{RepoAgentsDir: agentsDir}

	if err := InitProjectContext(repoDir, dirs); err != nil {
		t.Fatalf("InitProjectContext: %v", err)
	}
	if err := InitProjectContext(repoDir, dirs); err != nil {
		t.Fatalf("InitProjectContext second run: %v", err)
	}

	for _, rel := range []string{filepath.Join(".codebuddy", "skills"), filepath.Join(".claude", "skills")} {
		path := filepath.Join(repoDir, rel)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("lstat %s: %v", rel, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is not a symlink", rel)
		}
		target, err := os.Readlink(path)
		if err != nil {
			t.Fatalf("readlink %s: %v", rel, err)
		}
		if !sameCleanPath(target, filepath.Join(agentsDir, "shared-skills")) {
			t.Fatalf("%s target = %s", rel, target)
		}
	}

	excludePath := filepath.Join(repoDir, ".git", "info", "exclude")
	data, err := os.ReadFile(excludePath)
	if err != nil {
		t.Fatalf("read exclude: %v", err)
	}
	content := string(data)
	if strings.Count(content, ".codebuddy/") != 1 {
		t.Fatalf("exclude content missing .codebuddy/ exactly once: %q", content)
	}
	if strings.Count(content, ".claude/") != 1 {
		t.Fatalf("exclude content missing .claude/ exactly once: %q", content)
	}
}

func TestInitProjectContextSupportsWorktreeGitFile(t *testing.T) {
	base := t.TempDir()
	repoDir := filepath.Join(base, "repo")
	gitDir := filepath.Join(base, "gitdir")
	if err := os.MkdirAll(filepath.Join(repoDir), 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatalf("mkdir gitdir/info: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".git"), []byte("gitdir: ../gitdir\n"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}
	agentsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentsDir, "shared-skills", "issue-plan"), 0o755); err != nil {
		t.Fatalf("mkdir shared-skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "shared-skills", "issue-plan", "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("write shared skill: %v", err)
	}

	if err := InitProjectContext(repoDir, DataDirs{RepoAgentsDir: agentsDir}); err != nil {
		t.Fatalf("InitProjectContext: %v", err)
	}

	excludePath := filepath.Join(gitDir, "info", "exclude")
	data, err := os.ReadFile(excludePath)
	if err != nil {
		t.Fatalf("read exclude: %v", err)
	}
	if !strings.Contains(string(data), ".codebuddy/") || !strings.Contains(string(data), ".claude/") {
		t.Fatalf("exclude content = %q", string(data))
	}
}

func TestLoadDefaultInstructionsSelectsByCommand(t *testing.T) {
	agentsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agentsDir, "chat"), 0o755); err != nil {
		t.Fatalf("mkdir chat: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "chat", "CODEBUDDY.md"), []byte("codebuddy rules"), 0o644); err != nil {
		t.Fatalf("write CODEBUDDY.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "chat", "CLAUDE.md"), []byte("claude rules"), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	dirs := DataDirs{RepoAgentsDir: agentsDir}

	if got := loadDefaultInstructions(dirs, "chat", "/usr/local/bin/codebuddy"); !strings.Contains(got, "codebuddy rules") {
		t.Fatalf("codebuddy instructions = %q", got)
	}
	if got := loadDefaultInstructions(dirs, "chat", "/usr/local/bin/claude"); !strings.Contains(got, "claude rules") {
		t.Fatalf("claude instructions = %q", got)
	}
	if got := loadDefaultInstructions(dirs, "chat", "/usr/bin/python3"); got != "" {
		t.Fatalf("unexpected instructions for unrelated command: %q", got)
	}
}
