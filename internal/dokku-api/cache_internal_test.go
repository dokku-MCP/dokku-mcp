package dokkuApi

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *CommandCacheManager {
	t.Helper()
	cm := NewCommandCacheManager(&CacheConfig{Enabled: true, DefaultTTL: time.Minute}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(cm.Stop)
	return cm
}

func TestCacheDropsResultsComputedBeforeInvalidation(t *testing.T) {
	cm := newTestCache(t)

	generation := cm.Generation()
	// A write invalidates the cache while the read is still running.
	cm.Invalidate()
	cm.Set("apps:list", nil, []byte("stale"), generation)

	if _, found := cm.Get("apps:list", nil); found {
		t.Fatal("a result computed before an invalidation must not be cached")
	}

	cm.Set("apps:list", nil, []byte("fresh"), cm.Generation())
	if got, found := cm.Get("apps:list", nil); !found || string(got) != "fresh" {
		t.Fatalf("Get = %q, %v", got, found)
	}
}

func TestCacheKeySeparatesArguments(t *testing.T) {
	cm := newTestCache(t)
	if cm.generateCacheKey("config:show", []string{"ab", "c"}) == cm.generateCacheKey("config:show", []string{"a", "bc"}) {
		t.Fatal("different argument lists must not share a cache key")
	}
}
