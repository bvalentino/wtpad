package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBranchStandardRepo(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)

	got := DetectBranch(dir)
	if got != "main" {
		t.Errorf("DetectBranch() = %q, want %q", got, "main")
	}
}

func TestDetectBranchFeatureBranch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature/cool-thing\n"), 0o644)

	got := DetectBranch(dir)
	if got != "feature/cool-thing" {
		t.Errorf("DetectBranch() = %q, want %q", got, "feature/cool-thing")
	}
}

func TestDetectBranchDetachedHEAD(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	sha := "abc1234567890def"
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(sha+"\n"), 0o644)

	got := DetectBranch(dir)
	if got != "abc1234" {
		t.Errorf("DetectBranch() = %q, want %q", got, "abc1234")
	}
}

func TestDetectBranchLinkedWorktree(t *testing.T) {
	// Set up a fake main repo
	mainDir := t.TempDir()
	mainGitDir := filepath.Join(mainDir, ".git")
	os.MkdirAll(filepath.Join(mainGitDir, "worktrees", "wt1"), 0o755)
	os.WriteFile(filepath.Join(mainGitDir, "worktrees", "wt1", "HEAD"),
		[]byte("ref: refs/heads/wt-branch\n"), 0o644)

	// Set up linked worktree directory
	wtDir := t.TempDir()
	gitdirPath := filepath.Join(mainGitDir, "worktrees", "wt1")
	os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: "+gitdirPath+"\n"), 0o644)

	got := DetectBranch(wtDir)
	if got != "wt-branch" {
		t.Errorf("DetectBranch() = %q, want %q", got, "wt-branch")
	}
}

func TestFindGitDirStandardRepo(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)

	got := FindGitDir(dir)
	if got != gitDir {
		t.Errorf("FindGitDir() = %q, want %q", got, gitDir)
	}
}

func TestFindGitDirLinkedWorktree(t *testing.T) {
	// Set up a fake main repo with a worktree
	mainDir := t.TempDir()
	mainGitDir := filepath.Join(mainDir, ".git")
	os.MkdirAll(filepath.Join(mainGitDir, "worktrees", "wt1"), 0o755)

	// Set up linked worktree directory
	wtDir := t.TempDir()
	gitdirPath := filepath.Join(mainGitDir, "worktrees", "wt1")
	os.WriteFile(filepath.Join(wtDir, ".git"),
		[]byte("gitdir: "+gitdirPath+"\n"), 0o644)

	got := FindGitDir(wtDir)
	if got != mainGitDir {
		t.Errorf("FindGitDir() = %q, want %q", got, mainGitDir)
	}
}

func TestFindGitDirNonGitDir(t *testing.T) {
	dir := t.TempDir()
	got := FindGitDir(dir)
	if got != "" {
		t.Errorf("FindGitDir() = %q, want empty string", got)
	}
}

func TestDetectBranchNonGitDir(t *testing.T) {
	dir := t.TempDir()
	got := DetectBranch(dir)
	if got != "" {
		t.Errorf("DetectBranch() = %q, want empty string", got)
	}
}

func TestDetectBranchSubdirectory(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)

	subDir := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(subDir, 0o755)

	got := DetectBranch(subDir)
	if got != "main" {
		t.Errorf("DetectBranch() = %q, want %q", got, "main")
	}
}
