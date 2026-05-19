package discovery

import (
	"testing"
	"time"
)

func clearGlobalProcessCache() {
	globalProcessCache.mu.Lock()
	defer globalProcessCache.mu.Unlock()
	globalProcessCache.data = make(map[string]ProcessCacheEntry)
}

func TestMakeProcessKey(t *testing.T) {
	tests := []struct {
		pid        int32
		createTime int64
		want       string
	}{
		{1234, 1000000, "1234-1000000"},
		{1, 0, "1-0"},
		{0, 0, "0-0"},
	}

	for _, tt := range tests {
		got := makeProcessKey(tt.pid, tt.createTime)
		if got != tt.want {
			t.Errorf("makeProcessKey(%d, %d) = %q, want %q", tt.pid, tt.createTime, got, tt.want)
		}
	}
}

func TestCacheProcessMetadata_PutGet(t *testing.T) {
	clearGlobalProcessCache()

	entry := ProcessCacheEntry{
		ServiceName: "billing-api",
		ServiceType: "systemd",
		Owner:       "deploy",
	}

	CacheProcessMetadata(100, 5000, entry)

	got, ok := GetCachedProcessMetadata(100, 5000)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.ServiceName != "billing-api" {
		t.Errorf("ServiceName = %q, want %q", got.ServiceName, "billing-api")
	}
	if got.ServiceType != "systemd" {
		t.Errorf("ServiceType = %q, want %q", got.ServiceType, "systemd")
	}
	if got.Owner != "deploy" {
		t.Errorf("Owner = %q, want %q", got.Owner, "deploy")
	}
	if got.LastSeen == 0 {
		t.Error("LastSeen should be set on cache hit")
	}
}

func TestCacheProcessMetadata_Miss(t *testing.T) {
	clearGlobalProcessCache()

	_, ok := GetCachedProcessMetadata(999, 1000)
	if ok {
		t.Error("expected cache miss for uncached PID")
	}
}

func TestCacheProcessMetadata_DifferentCreateTime(t *testing.T) {
	clearGlobalProcessCache()

	CacheProcessMetadata(100, 5000, ProcessCacheEntry{ServiceName: "old-proc"})

	_, ok := GetCachedProcessMetadata(100, 6000)
	if ok {
		t.Error("same PID with different createTime should miss")
	}

	got, ok := GetCachedProcessMetadata(100, 5000)
	if !ok {
		t.Fatal("original key should still hit")
	}
	if got.ServiceName != "old-proc" {
		t.Errorf("ServiceName = %q, want %q", got.ServiceName, "old-proc")
	}
}

func TestCacheProcessMetadata_Overwrite(t *testing.T) {
	clearGlobalProcessCache()

	CacheProcessMetadata(100, 5000, ProcessCacheEntry{ServiceName: "v1"})
	CacheProcessMetadata(100, 5000, ProcessCacheEntry{ServiceName: "v2"})

	got, ok := GetCachedProcessMetadata(100, 5000)
	if !ok {
		t.Fatal("expected cache hit after overwrite")
	}
	if got.ServiceName != "v2" {
		t.Errorf("ServiceName = %q, want %q (latest write wins)", got.ServiceName, "v2")
	}
}

func TestCacheProcessMetadata_LastSeenUpdatedOnRead(t *testing.T) {
	clearGlobalProcessCache()

	CacheProcessMetadata(100, 5000, ProcessCacheEntry{ServiceName: "billing"})

	got1, _ := GetCachedProcessMetadata(100, 5000)
	ts1 := got1.LastSeen

	time.Sleep(time.Millisecond)

	got2, _ := GetCachedProcessMetadata(100, 5000)
	if got2.LastSeen < ts1 {
		t.Error("LastSeen should not decrease on subsequent reads")
	}
}

func TestPruneProcessCache(t *testing.T) {
	clearGlobalProcessCache()

	CacheProcessMetadata(1, 100, ProcessCacheEntry{ServiceName: "keep-me"})

	globalProcessCache.mu.Lock()
	globalProcessCache.data["2-200"] = ProcessCacheEntry{
		ServiceName: "stale",
		LastSeen:    time.Now().Add(-30 * time.Minute).Unix(),
	}
	globalProcessCache.mu.Unlock()

	PruneProcessCache()

	if _, ok := GetCachedProcessMetadata(1, 100); !ok {
		t.Error("fresh entry should survive pruning")
	}

	globalProcessCache.mu.Lock()
	_, staleExists := globalProcessCache.data["2-200"]
	globalProcessCache.mu.Unlock()
	if staleExists {
		t.Error("stale entry (30 min old) should be pruned (threshold is 20 min)")
	}
}

func TestPruneProcessCache_EmptyIsNoop(t *testing.T) {
	clearGlobalProcessCache()
	PruneProcessCache()

	globalProcessCache.mu.Lock()
	count := len(globalProcessCache.data)
	globalProcessCache.mu.Unlock()
	if count != 0 {
		t.Errorf("expected empty cache after pruning empty cache, got %d entries", count)
	}
}

func TestCacheProcessMetadata_ContainerInfo(t *testing.T) {
	clearGlobalProcessCache()

	ci := &ContainerInfo{
		IsContainer: true,
		ContainerID: "abc123",
		Runtime:     "docker",
	}
	CacheProcessMetadata(100, 5000, ProcessCacheEntry{
		ServiceName:   "containerized",
		ContainerInfo: ci,
	})

	got, ok := GetCachedProcessMetadata(100, 5000)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.ContainerInfo == nil {
		t.Fatal("ContainerInfo should be preserved in cache")
	}
	if got.ContainerInfo.ContainerID != "abc123" {
		t.Errorf("ContainerID = %q, want %q", got.ContainerInfo.ContainerID, "abc123")
	}
}
