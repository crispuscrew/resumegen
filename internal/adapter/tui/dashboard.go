//go:build !notui

package tui

import (
	"fmt"
	"strings"

	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// dashboardView renders counts computed by track.Summarize. It reads no state
// the use case didn't hand it - the bucketing lives in the gated layer.
func (m model) dashboardView() string {
	if m.loading {
		return m.styl.subtle.Render("loading…")
	}
	if m.loadErr != nil {
		return m.styl.errText.Render("error: " + m.loadErr.Error())
	}
	if len(m.apps) == 0 {
		return m.styl.subtle.Render("no applications yet - press n to create one.")
	}

	s := track.Summarize(m.apps, m.deps.Tracker.Now(), m.deps.StaleAfterDays)

	var b strings.Builder
	fmt.Fprintf(&b, "%s   %d total   ·   %s active   ·   %d closed\n",
		m.styl.label.Render("Applications"), s.Total,
		m.styl.title.Render(fmt.Sprintf("%d", s.Active)), s.Terminal)

	if s.FollowupsDue > 0 || s.StaleActive > 0 {
		b.WriteString("\n")
		if s.FollowupsDue > 0 {
			b.WriteString(m.styl.badge.Render(fmt.Sprintf("  ● %d followup(s) due", s.FollowupsDue)) + "\n")
		}
		if s.StaleActive > 0 {
			b.WriteString(m.styl.badge.Render(fmt.Sprintf("  ● %d quiet >=%dd (auto-ghost at %dd)", s.StaleActive, m.deps.StaleAfterDays, m.deps.GhostAfterDays)) + "\n")
		}
	}

	b.WriteString("\n" + m.styl.label.Render("By status") + "\n")
	for _, st := range track.StatusesInPipelineOrder() {
		n := s.ByStatus[st]
		if n == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %-11s %d\n", st, n)
	}
	return b.String()
}
