package approval

import (
	"context"
	"testing"
)

func TestAllowAndAllowed(t *testing.T) {
	ctx := WithTracker(context.Background())

	if Allowed(ctx, "d1") {
		t.Fatal("document should not be allowed initially")
	}
	Allow(ctx, "d1")
	if !Allowed(ctx, "d1") {
		t.Fatal("document should be allowed after Allow")
	}
	if Allowed(ctx, "d2") {
		t.Fatal("other document should not be allowed")
	}
}

// TestAllowWithoutTracker ctx 上没有追踪器时静默忽略, 不 panic
func TestAllowWithoutTracker(t *testing.T) {
	ctx := context.Background()
	Allow(ctx, "d1")
	if Allowed(ctx, "d1") {
		t.Fatal("document should not be allowed without tracker")
	}
}

// TestWithTrackerReuse 已有追踪器时复用, 不丢失已授权文档
func TestWithTrackerReuse(t *testing.T) {
	ctx := WithTracker(context.Background())
	Allow(ctx, "d1")

	reused := WithTracker(ctx)
	if !Allowed(reused, "d1") {
		t.Fatal("existing tracker should be reused")
	}
}
