package main

import (
	"testing"

	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
)

func TestFindNewPublishedFeedPrefersExactTitle(t *testing.T) {
	before := snapshotFeeds([]xiaohongshu.Feed{
		{ID: "old-1", NoteCard: xiaohongshu.NoteCard{DisplayTitle: "older"}},
	})

	after := []xiaohongshu.Feed{
		{ID: "new-1", XsecToken: "token-1", NoteCard: xiaohongshu.NoteCard{DisplayTitle: "another"}},
		{ID: "new-2", XsecToken: "token-2", NoteCard: xiaohongshu.NoteCard{DisplayTitle: "target title"}},
	}

	feed := findNewPublishedFeed(before, after, "target title")
	if feed == nil {
		t.Fatal("expected new feed")
	}
	if feed.ID != "new-2" {
		t.Fatalf("expected exact-title match, got %s", feed.ID)
	}
}

func TestFindNewPublishedFeedFallsBackToFirstNewFeed(t *testing.T) {
	before := snapshotFeeds([]xiaohongshu.Feed{{ID: "old-1"}})
	after := []xiaohongshu.Feed{
		{ID: "new-1", XsecToken: "token-1", NoteCard: xiaohongshu.NoteCard{DisplayTitle: "first"}},
		{ID: "new-2", XsecToken: "token-2", NoteCard: xiaohongshu.NoteCard{DisplayTitle: "second"}},
	}

	feed := findNewPublishedFeed(before, after, "missing title")
	if feed == nil {
		t.Fatal("expected fallback feed")
	}
	if feed.ID != "new-1" {
		t.Fatalf("expected first new feed, got %s", feed.ID)
	}
}
