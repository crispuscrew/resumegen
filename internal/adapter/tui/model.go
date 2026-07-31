//go:build !notui

package tui

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// screen is the active view. Dashboard, applications, generate, prompts, data,
// and config are the number-key destinations; detail is a sub-screen reached by
// drilling into an application.
type screen int

const (
	screenDashboard screen = iota
	screenList
	screenDetail
	screenGenerate
	screenPrompts
	screenData
	screenConfig
	screenNewApp // creation form, reached with `n` from dashboard/list
)

// errNoCapability is returned when a screen's injected function is nil (only
// possible in a misconfigured build; the cli always wires them).
var errNoCapability = errors.New("unavailable in this build")

// model is the root bubbletea model. It owns the loaded application set plus the
// per-screen state. Every decision it renders is computed by track/domain or by
// an injected cli function; the model only routes keys and paints results.
type model struct {
	deps   Deps
	styl   styles
	width  int
	height int

	active   screen
	apps     []domain.Application
	loading  bool
	loadErr  error
	showHelp bool
	quitting bool

	cursor   int         // list selection (indexes the filtered view)
	detail   detailState // detail sub-screen
	generate generateState
	prompts  promptsState
	data     dataState
	config   configState
	newapp   newAppState

	// applications list filter (the `/` search).
	filter      string
	filtering   bool
	filterInput textinput.Model
	// flash is a transient status line (e.g. a yank confirmation), cleared on
	// screen switch.
	flash string
}

// --- shared messages / commands (tracker) ---

type appsLoadedMsg struct {
	apps []domain.Application
	err  error
}

type actionDoneMsg struct {
	app   domain.Application
	err   error
	flash string
}

func (m model) loadApps() tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		apps, err := tr.List(ctx, track.Query{})
		return appsLoadedMsg{apps: apps, err: err}
	}
}

func (m model) transitionCmd(id string, to domain.Status) tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		app, err := tr.Transition(ctx, id, to, "")
		return actionDoneMsg{app: app, err: err, flash: "status -> " + string(to)}
	}
}

func (m model) noteCmd(id, text string) tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		app, err := tr.AddNote(ctx, id, text)
		return actionDoneMsg{app: app, err: err, flash: "note added"}
	}
}

func (m model) followupCmd(id string, due time.Time, action string) tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		app, err := tr.AddFollowup(ctx, id, due, action)
		return actionDoneMsg{app: app, err: err, flash: "followup added"}
	}
}

func (m model) completeFollowupCmd(id string, index int) tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		app, err := tr.CompleteFollowup(ctx, id, index)
		return actionDoneMsg{app: app, err: err, flash: fmt.Sprintf("followup %d done", index)}
	}
}

// --- bubbletea.Model ---

func (m model) Init() tea.Cmd { return m.loadApps() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)

	case appsLoadedMsg:
		m.loading = false
		m.loadErr = msg.err
		if msg.err == nil {
			m.apps = msg.apps
			// Newest first for humans: IDs start with the creation date, so a
			// descending ID sort is a descending date sort. (The CLI keeps
			// ascending order - its output is a stable scripting contract.)
			sort.Slice(m.apps, func(i, j int) bool { return m.apps[i].ID > m.apps[j].ID })
			// The cursor indexes the FILTERED view, so clamp against it - a
			// reload can shrink the visible set while a filter is active.
			if vis := m.visibleApps(); m.cursor >= len(vis) {
				m.cursor = maxInt(0, len(vis)-1)
			}
			if m.active == screenDetail {
				if a, ok := findByID(m.apps, m.detail.app.ID); ok {
					m.detail.app = a
				}
			}
		}
		return m, nil
	case actionDoneMsg:
		// Always clear busy: the action that set it has landed either way.
		m.detail.busy = false
		// Drop a result that no longer belongs to what is on screen. Without
		// this, an action dispatched on one application and completing after
		// the user navigated to another would overwrite the detail view with
		// the first application and show its flash there. Mirrors the identity
		// guards in onPromptForm/onPromptRun/onNewAppDone.
		if m.active != screenDetail || (msg.err == nil && msg.app.ID != m.detail.app.ID) {
			return m, m.loadApps()
		}
		if msg.err != nil {
			m.detail.flash = ""
			m.detail.errMsg = msg.err.Error()
			m.detail.mode = detailView
			return m, nil
		}
		m.detail.app = msg.app
		m.detail.mode = detailView
		m.detail.flash = msg.flash
		m.detail.errMsg = ""
		return m, m.loadApps()

	case newAppDoneMsg:
		return m.onNewAppDone(msg)

	// generate
	case profilesLoadedMsg:
		return m.onProfilesLoaded(msg)
	case renderDoneMsg:
		return m.onRenderDone(msg)
	// prompts
	case promptsLoadedMsg:
		return m.onPromptsLoaded(msg)
	case promptFormMsg:
		return m.onPromptForm(msg)
	case promptRunMsg:
		return m.onPromptRun(msg)
	case copyDoneMsg:
		return m.onCopyDone(msg)
	// data
	case dataFilesMsg:
		return m.onDataFiles(msg)
	case editorFinishedMsg:
		return m.onEditorFinished(msg)
	// config
	case configLoadedMsg:
		return m.onConfigLoaded(msg)
	}
	return m, nil
}

// handleKey applies global keys, then routes to the active screen. When a screen
// is capturing text (detail note/followup, prompt field), the key goes straight
// to that screen so typed "q"/"?"/digits aren't stolen as navigation.
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		if m.generate.cancel != nil {
			m.generate.cancel() // don't orphan an in-flight typst on force quit
		}
		m.quitting = true
		return m, tea.Quit
	}
	if m.isCapturing() {
		return m.routeKey(msg)
	}
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	switch msg.String() {
	case "?":
		m.showHelp = true
		return m, nil
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "1":
		return m.enter(screenDashboard)
	case "2":
		return m.enter(screenList)
	case "3":
		return m.enter(screenGenerate)
	case "4":
		return m.enter(screenPrompts)
	case "5":
		return m.enter(screenData)
	case "6":
		return m.enter(screenConfig)
	}
	return m.routeKey(msg)
}

func (m model) routeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.active {
	case screenDashboard:
		return m.updateDashboard(msg)
	case screenList:
		return m.updateList(msg)
	case screenDetail:
		return m.updateDetail(msg)
	case screenGenerate:
		return m.updateGenerate(msg)
	case screenPrompts:
		return m.updatePrompts(msg)
	case screenData:
		return m.updateData(msg)
	case screenConfig:
		return m.updateConfig(msg)
	case screenNewApp:
		return m.updateNewApp(msg)
	}
	return m, nil
}

// isCapturing reports whether the active screen should take keys directly rather
// than letting them act as global navigation: a text-entry sub-mode, or a render
// in flight (so a digit can't navigate away and orphan the running typst).
func (m model) isCapturing() bool {
	switch m.active {
	case screenDetail:
		return m.detail.capturing()
	case screenPrompts:
		return m.prompts.capturing()
	case screenGenerate:
		return m.generate.phase == genRunning
	case screenList:
		return m.filtering
	case screenNewApp:
		return true // the whole screen is a form
	}
	return false
}

// switchTo changes the active screen and clears the transient flash - a
// confirmation belongs to the screen it was raised on. Every screen change
// funnels through here or enter().
func (m model) switchTo(s screen) model {
	m.active = s
	m.flash = ""
	return m
}

// enter switches to a top-level screen and returns its lazy-load command.
func (m model) enter(s screen) (tea.Model, tea.Cmd) {
	m = m.switchTo(s)
	switch s {
	case screenGenerate:
		return m, m.loadProfiles()
	case screenPrompts:
		m.prompts.phase = promptsPickList // fresh entry always starts at the picker
		return m, m.loadPrompts()
	case screenData:
		return m, m.loadDataFiles()
	case screenConfig:
		return m, m.loadConfig()
	}
	return m, nil
}

func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "tab", "l", "right":
		m = m.switchTo(screenList)
		return m, nil
	case "n":
		return m.openNewApp()
	case "r":
		m.loading = true
		return m, m.loadApps()
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.updateListFilter(msg)
	}
	vis := m.visibleApps()
	switch msg.String() {
	case "esc":
		if m.filter != "" { // first esc clears an active filter, next leaves
			m.filter = ""
			m.cursor = 0
			return m, nil
		}
		m = m.switchTo(screenDashboard)
		return m, nil
	case "tab":
		m = m.switchTo(screenDashboard)
		return m, nil
	case "n":
		return m.openNewApp()
	case "/":
		m.filtering = true
		m.filterInput = newTextInput("filter by company / role / id / status")
		m.filterInput.SetValue(m.filter)
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(vis)-1 {
			m.cursor++
		}
		return m, nil
	case "r":
		m.loading = true
		return m, m.loadApps()
	case "y":
		if len(vis) == 0 {
			return m, nil
		}
		id := vis[m.cursor].ID
		return m, m.copyCmd(id, "copied id: "+id)
	case "enter", "l", "right":
		if len(vis) == 0 {
			return m, nil
		}
		m.detail = detailState{app: vis[m.cursor], mode: detailView}
		m = m.switchTo(screenDetail)
		return m, nil
	}
	return m, nil
}

// updateListFilter handles keys while the `/` filter input is focused. Typing
// filters live; enter confirms (keeps the filter), esc cancels (clears it).
func (m model) updateListFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.cursor = 0
		return m, nil
	case "enter":
		m.filter = strings.TrimSpace(m.filterInput.Value())
		m.filtering = false
		m.cursor = 0
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = strings.TrimSpace(m.filterInput.Value())
	if c := len(m.visibleApps()); m.cursor >= c {
		m.cursor = maxInt(0, c-1)
	}
	return m, cmd
}

// visibleApps is the application set after the `/` filter is applied.
func (m model) visibleApps() []domain.Application { return filterApps(m.apps, m.filter) }

// filterApps keeps applications whose company/role/id/status contains q
// (case-insensitive). An empty query returns all. Pure and testable.
func filterApps(apps []domain.Application, q string) []domain.Application {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return apps
	}
	out := make([]domain.Application, 0, len(apps))
	for _, a := range apps {
		hay := strings.ToLower(a.Company + " " + a.Role + " " + a.ID + " " + string(a.Status))
		if strings.Contains(hay, q) {
			out = append(out, a)
		}
	}
	return out
}

// --- view ---

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.showHelp {
		return m.chrome(m.helpView())
	}
	var body string
	switch m.active {
	case screenDashboard:
		body = m.dashboardView()
	case screenList:
		body = m.listView()
	case screenDetail:
		body = m.detailView()
	case screenGenerate:
		body = m.generateView()
	case screenPrompts:
		body = m.promptsView()
	case screenData:
		body = m.dataView()
	case screenConfig:
		body = m.configView()
	case screenNewApp:
		body = m.newAppView()
	}
	return m.chrome(body)
}

func (m model) chrome(body string) string {
	nav := m.navBar()
	footer := m.styl.footer.Render(m.footerHints())
	return lipgloss.JoinVertical(lipgloss.Left, nav, "", body, "", footer)
}

// navBar renders the numbered top-level destinations, highlighting the active.
func (m model) navBar() string {
	items := []struct {
		s     screen
		label string
	}{
		{screenDashboard, "1 dashboard"},
		{screenList, "2 applications"},
		{screenGenerate, "3 generate"},
		{screenPrompts, "4 prompts"},
		{screenData, "5 data"},
		{screenConfig, "6 config"},
	}
	parts := make([]string, 0, len(items))
	for _, it := range items {
		active := m.active == it.s || (it.s == screenList && (m.active == screenDetail || m.active == screenNewApp))
		if active {
			parts = append(parts, m.styl.selected.Render(" "+it.label+" "))
		} else {
			parts = append(parts, m.styl.subtle.Render(" "+it.label+" "))
		}
	}
	title := m.styl.title.Render("resumegen ") + m.styl.subtle.Render(m.deps.Version)
	return lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(parts, m.styl.subtle.Render("·")))
}

func (m model) footerHints() string {
	switch m.active {
	case screenDashboard:
		return "enter: applications · n: new · 1-6: switch · r: refresh · ?: help · q: quit"
	case screenList:
		if m.filtering {
			return "type to filter · enter: apply · esc: clear"
		}
		return "↑/↓: move · enter: open · n: new · /: filter · y: copy id · esc: back · q: quit"
	case screenDetail:
		return m.detailFooter()
	case screenGenerate:
		return m.generateFooter()
	case screenPrompts:
		return m.promptsFooter()
	case screenData:
		return "↑/↓: move · enter: edit in $EDITOR · esc: dashboard · 1-6: switch · q: quit"
	case screenConfig:
		return "r: reload · esc: dashboard · 1-6: switch · ?: help · q: quit"
	case screenNewApp:
		return m.newAppFooter()
	}
	return ""
}

func (m model) helpView() string {
	lines := []string{
		m.styl.label.Render("Keys"),
		"",
		"  Global",
		"    1-6          switch screen (dashboard/applications/generate/prompts/data/config)",
		"    ?            toggle this help      q  quit      ctrl+c  force quit",
		"",
		"  Applications",
		"    ↑/↓ or j/k    move · enter open · esc back",
		"    n            new application (also works on the dashboard)",
		"    /            filter · y copy id to clipboard",
		"    in detail:   s change status · n add note · f add followup · d complete followup",
		"",
		"  Generate",
		"    ↑/↓ pick profile · enter render · esc cancel a running render",
		"",
		"  Prompts",
		"    ↑/↓ pick template · enter fill · tab next field · enter run · y copy",
		"",
		"  Data",
		"    ↑/↓ pick file · enter edit in $EDITOR",
	}
	return strings.Join(lines, "\n")
}

// --- small helpers ---

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func findByID(apps []domain.Application, id string) (domain.Application, bool) {
	for _, a := range apps {
		if a.ID == id {
			return a, true
		}
	}
	return domain.Application{}, false
}

func (m model) field(label, value string) string {
	return "  " + m.styl.subtle.Render(fmt.Sprintf("%-9s", label)) + " " + value + "\n"
}

func (m model) fieldWide(label, value string) string {
	return "  " + m.styl.subtle.Render(fmt.Sprintf("%-32s", label)) + " " + value + "\n"
}
