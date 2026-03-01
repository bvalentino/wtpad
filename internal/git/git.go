package git

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectBranch returns the current branch name by reading .git/HEAD directly.
// For linked worktrees, it follows the gitdir pointer. Returns the short SHA
// for detached HEAD, or "" if not in a git repository.
func DetectBranch(cwd string) string {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}

	headPath := findHEAD(abs)
	if headPath == "" {
		return ""
	}

	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}

	return parseBranch(strings.TrimSpace(string(data)))
}

// FindGitDir walks up from dir looking for a .git directory. For linked
// worktrees where .git is a file, it follows the gitdir pointer back to the
// main repository's .git directory. Returns "" if not in a git repository.
func FindGitDir(dir string) string {
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil {
			if info.IsDir() {
				return gitPath
			}
			// Linked worktree: .git is a file containing "gitdir: <path>"
			// Follow the pointer to get the worktree gitdir, then walk up
			// to the main .git directory.
			if wd := resolveGitFile(gitPath); wd != "" {
				return findParentGitDir(wd)
			}
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findHEAD walks up from dir looking for .git (directory or file) and returns
// the path to the HEAD file, or "" if not found.
func findHEAD(dir string) string {
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil {
			if info.IsDir() {
				return filepath.Join(gitPath, "HEAD")
			}
			// Linked worktree: .git is a file — read HEAD from the gitdir
			if wd := resolveGitFile(gitPath); wd != "" {
				return filepath.Join(wd, "HEAD")
			}
			return ""
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// resolveGitFile reads a .git file (linked worktree) and returns the gitdir path.
func resolveGitFile(gitFile string) string {
	data, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return ""
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(gitFile), gitdir)
	}

	return gitdir
}

// findParentGitDir walks up from a worktree gitdir to find the main .git directory.
// e.g., /repo/.git/worktrees/wt1 → /repo/.git
func findParentGitDir(worktreeDir string) string {
	dir := worktreeDir
	for {
		if filepath.Base(dir) == ".git" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// parseBranch extracts the branch name from HEAD content.
func parseBranch(head string) string {
	if strings.HasPrefix(head, "ref: refs/heads/") {
		return strings.TrimPrefix(head, "ref: refs/heads/")
	}
	// Detached HEAD — return short SHA
	if len(head) >= 7 {
		return head[:7]
	}
	return ""
}
