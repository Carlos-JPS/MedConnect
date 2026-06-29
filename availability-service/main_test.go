package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildSlotDirectoryMapsSlotIDsToShard(t *testing.T) {
	directory, err := buildSlotDirectory(context.Background(), map[string]slotIDLister{
		"shard-a": fakeSlotIDLister{slotIDs: []string{"slot-1", "slot-2"}},
		"shard-b": fakeSlotIDLister{slotIDs: []string{"slot-3"}},
	})
	if err != nil {
		t.Fatalf("buildSlotDirectory returned error: %v", err)
	}

	want := map[string]string{
		"slot-1": "shard-a",
		"slot-2": "shard-a",
		"slot-3": "shard-b",
	}
	for slotID, shardName := range want {
		if got := directory[slotID]; got != shardName {
			t.Fatalf("expected %s -> %s, got %s", slotID, shardName, got)
		}
	}
}

func TestBuildSlotDirectoryRejectsDuplicateSlotIDs(t *testing.T) {
	_, err := buildSlotDirectory(context.Background(), map[string]slotIDLister{
		"shard-a": fakeSlotIDLister{slotIDs: []string{"slot-1"}},
		"shard-b": fakeSlotIDLister{slotIDs: []string{"slot-1"}},
	})
	if err == nil {
		t.Fatal("expected duplicate slot_id error")
	}
	if !strings.Contains(err.Error(), "slot-1") {
		t.Fatalf("expected error to mention duplicated slot_id, got %v", err)
	}
}

func TestBuildSlotDirectoryPropagatesListerError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	_, err := buildSlotDirectory(context.Background(), map[string]slotIDLister{
		"shard-a": fakeSlotIDLister{err: wantErr},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped lister error, got %v", err)
	}
}

type fakeSlotIDLister struct {
	slotIDs []string
	err     error
}

func (f fakeSlotIDLister) ListSlotIDs(ctx context.Context) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.slotIDs, nil
}
