package plugin

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// InitProjectContext ensures project-local agent context paths exist without
// polluting git status. The context is safe to apply repeatedly to both the
// main repo_local checkout and per-issue worktrees.
func InitProjectContext(repoDir string, dirs DataDirs) error {
	if repoDir == "" {
		return nil
	}
	sharedSkillsDir := filepath.Join(filepath.Dir(dirs.RepoAgentsDir), "skills")
	info, err := os.Stat(sharedSkillsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat shared skills dir: %w", err)
	}
	if !info.IsDir() {
		return nil
	}
	for _, rel := range []string{
		filepath.Join(".codebuddy", "skills"),
		filepath.Join(".claude", "skills"),
	} {
		if err := ensureProjectSymlink(filepath.Join(repoDir, rel), sharedSkillsDir); err != nil {
			return err
		}
	}
	if err := ensureGitExclude(repoDir, []string{".codebuddy/", ".claude/"}); err != nil {
		return err
	}
	return nil
}

func loadDefaultInstructions(dirs DataDirs, kind, command string) string {
	name := instructionFileName(command)
	if name == "" {
		return ""
	}
	path := filepath.Join(dirs.RepoAgentsDir, kind, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("read default instructions", "path", path, "err", err)
		}
		return ""
	}
	body := strings.TrimSpace(string(data))
	if body == "" {
		return ""
	}
	return fmt.Sprintf("Default %s instructions loaded from %s:\n\n%s", kind, path, body)
}

func instructionFileName(command string) string {
	switch strings.ToLower(filepath.Base(command)) {
	case "codebuddy":
		return "CODEBUDDY.md"
	case "claude":
		return "CLAUDE.md"
	default:
		return ""
	}
}

func ensureProjectSymlink(path, target string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent for %s: %w", path, err)
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			current, err := os.Readlink(path)
			if err == nil {
				if !filepath.IsAbs(current) {
					current = filepath.Join(filepath.Dir(path), current)
				}
				if sameCleanPath(current, target) {
					return nil
				}
			}
		}
		slog.Warn("project context path already exists; leaving it untouched", "path", path)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if err := os.Symlink(target, path); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", path, target, err)
	}
	return nil
}

func sameCleanPath(a, b string) bool {
	aAbs, err := filepath.Abs(a)
	if err != nil {
		aAbs = a
	}
	bAbs, err := filepath.Abs(b)
	if err != nil {
		bAbs = b
	}
	return filepath.Clean(aAbs) == filepath.Clean(bAbs)
}

func ensureGitExclude(repoDir string, patterns []string) error {
	gitDir, err := resolveGitDir(repoDir)
	if err != nil {
		return err
	}
	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return fmt.Errorf("mkdir git info dir: %w", err)
	}
	excludePath := filepath.Join(infoDir, "exclude")
	data, err := os.ReadFile(excludePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read git exclude: %w", err)
	}
	content := string(data)
	seen := map[string]struct{}{}
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seen[line] = struct{}{}
	}
	var missing []string
	for _, pattern := range patterns {
		if _, ok := seen[pattern]; ok {
			continue
		}
		missing = append(missing, pattern)
	}
	if len(missing) == 0 {
		return nil
	}
	var out strings.Builder
	out.WriteString(content)
	if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
		out.WriteString("\n")
	}
	for _, pattern := range missing {
		out.WriteString(pattern)
		out.WriteString("\n")
	}
	if err := os.WriteFile(excludePath, []byte(out.String()), 0o644); err != nil {
		return fmt.Errorf("write git exclude: %w", err)
	}
	return nil
}

func resolveGitDir(repoDir string) (string, error) {
	gitPath := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", gitPath, err)
	}
	if info.IsDir() {
		return gitPath, nil
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", gitPath, err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir:") {
		return "", fmt.Errorf("unsupported .git file format in %s", gitPath)
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoDir, gitDir)
	}
	return filepath.Clean(gitDir), nil
}
