package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBlogUsesGitAdditionDatesAndHidesGitDirectory(t *testing.T) {
	dir := t.TempDir()
	commitDay := 3
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		// 23:51 in -04:00 is the following day in UTC; display the author's day.
		date := time.Date(2026, 3, commitDay, 23, 51, 0, 0, time.FixedZone("", -4*60*60)).Format(time.RFC3339)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com", "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	runGit("init")
	if err := os.Mkdir(filepath.Join(dir, "Notes"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"old.md", "new.md"} {
		if err := os.WriteFile(filepath.Join(dir, "Notes", name), []byte("content for "+name), 0644); err != nil {
			t.Fatal(err)
		}
		runGit("add", "Notes/"+name)
		runGit("-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "Add "+name)
		commitDay++
	}
	// Editing the older article must not move it ahead of the newer one.
	if err := os.WriteFile(filepath.Join(dir, "Notes", "old.md"), []byte("edited"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "Notes/old.md")
	runGit("commit", "-m", "Edit old.md")
	commitDay++
	runGit("mv", "Notes/old.md", "Notes/renamed.md")
	runGit("commit", "-m", "Rename old.md")
	for _, name := range []string{"renamed.md", "new.md"} {
		stamp := time.Now().Add(-time.Hour)
		if err := os.Chtimes(filepath.Join(dir, "Notes", name), stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	blog, err := NewBlog(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries := blog.Entries()
	if len(entries) != 1 || entries[0].Title != "Notes" || len(entries[0].Children) != 2 {
		t.Fatalf("unexpected entries: %+v", entries)
	}
	if entries[0].Children[0].Title != "new" || entries[0].Children[1].Title != "renamed" {
		t.Fatalf("unexpected note order: %+v", entries[0].Children)
	}
	article, err := blog.ArticleHTML("Notes/renamed")
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := article.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "March 3, 2026") {
		t.Fatal("article did not display its Git addition date")
	}
	article, err = blog.ArticleHTML("Notes/new")
	if err != nil {
		t.Fatal(err)
	}
	rendered.Reset()
	if err := article.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "March 4, 2026") {
		t.Fatal("new article did not display its original local calendar day")
	}
}
