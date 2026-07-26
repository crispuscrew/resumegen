package trackrepo

import (
	"context"
	"strings"
	"testing"
)

// The read path takes raw CLI ids; a crafted one must not escape applications/.
func TestHostileIDsRejected(t *testing.T) {
	s := New(t.TempDir())
	for _, id := range []string{"../../etc/passwd", "..", `a\b`, "a/b", ""} {
		if _, err := s.Load(context.Background(), id); err == nil || !strings.Contains(err.Error(), "invalid application id") {
			t.Errorf("Load(%q) should reject as invalid id, got %v", id, err)
		}
		if _, err := s.Exists(context.Background(), id); err == nil {
			t.Errorf("Exists(%q) should reject", id)
		}
		if err := s.Delete(context.Background(), id); err == nil {
			t.Errorf("Delete(%q) should reject", id)
		}
	}
}
