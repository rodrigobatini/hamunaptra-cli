package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rodrigobatini/hamunaptra-cli/internal/api"
	"github.com/rodrigobatini/hamunaptra-cli/internal/configfile"
	"github.com/rodrigobatini/hamunaptra-cli/internal/localproj"
)

type slashCommand struct {
	Name string
	Desc string
}

var slashCommands = []slashCommand{
	{Name: "commands", Desc: "Show available slash commands"},
	{Name: "integrations", Desc: "Show integration status"},
	{Name: "provider", Desc: "Run provider flow helper (ex: /provider vercel report)"},
	{Name: "pipeline", Desc: "Show guided readiness flow (ex: /pipeline vercel)"},
	{Name: "login", Desc: "Authenticate via browser device flow"},
	{Name: "init", Desc: "Create cloud project and link folder"},
	{Name: "connect", Desc: "Connect a provider (ex: vercel)"},
	{Name: "sync", Desc: "Sync provider cost snapshots"},
	{Name: "report", Desc: "Show project cost report"},
	{Name: "anomalies", Desc: "Show detected anomalies"},
	{Name: "ask", Desc: "Ask a cost question"},
	{Name: "help", Desc: "Show full CLI help"},
}

var shellAllowlist = map[string]bool{
	"pwd":    true,
	"ls":     true,
	"git":    true,
	"cat":    true,
	"echo":   true,
	"whoami": true,
	"date":   true,
}

type statusData struct {
	LoggedIn   bool
	APIBase    string
	Project    string
	ConfigPath string
	WorkDir    string
	VercelConnected bool
	VercelSource    string
	VercelLastSync  string
	VercelLastError string
}

type runResultMsg struct {
	output string
	err    error
}

type refreshStatusMsg struct{}

type model struct {
	width  int
	height int

	status statusData

	runCLIFn     func(args []string) tea.Cmd
	runShellFn   func(bin string, args []string) tea.Cmd
	lookPathFn   func(file string) (string, error)
	nowFn        func() time.Time

	input       string
	suggestions []int
	selectedSug int
	running     bool
	pinIntegrations bool
	postProcessMode string

	lastAction     string
	lastOutput     string
	quitArmedUntil time.Time
}

func Run() error {
	m := model{
		status:     collectStatus(),
		runCLIFn:   runCLICommandCmd,
		runShellFn: runShellCommandCmd,
		lookPathFn: exec.LookPath,
		nowFn:      time.Now,
		lastAction: "Ready.",
		lastOutput: strings.Join([]string{
			"Welcome to Hamunaptra TUI",
			"Hamunaptra is a terminal-first copilot for cloud cost visibility, anomaly spotting, and quick CLI actions.",
			"",
			"Quick start",
			"  /commands       list available slash commands",
			"  /integrations   show provider status",
			"  /login          authenticate",
			"  !ls             run a safe shell command",
			"",
			"Tips",
			"  Ask naturally and press Enter to run `hamunaptra ask ...`",
			"  Use ↑/↓ + Tab with slash suggestions",
		}, "\n"),
	}
	m.updateSuggestions()
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return refreshStatusCmd()
}

func refreshStatusCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return refreshStatusMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m = m.withRuntimeDefaults()
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case refreshStatusMsg:
		m.status = collectStatus()
		return m, refreshStatusCmd()
	case runResultMsg:
		m.running = false
		mode := m.postProcessMode
		m.postProcessMode = ""
		if msg.err != nil {
			m.lastAction = "Failed: " + humanizeRunError(msg.err.Error())
			m.lastOutput = humanizeRunError(msg.err.Error())
		} else {
			m.lastAction = "Done."
			out := strings.TrimSpace(msg.output)
			if out == "" {
				out = "(no output)"
			}
			if mode == "report_json" {
				out = summarizeReportOutput(out)
			}
			m.lastOutput = shortMultiline(out, 48)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			now := m.nowFn()
			if now.Before(m.quitArmedUntil) {
				return m, tea.Quit
			}
			m.quitArmedUntil = now.Add(2 * time.Second)
			m.lastAction = "Exit armed."
			m.lastOutput = "Press Ctrl+C again to quit."
			return m, nil
		case "up":
			if m.running || len(m.suggestions) == 0 {
				return m, nil
			}
			if m.selectedSug > 0 {
				m.selectedSug--
			}
			return m, nil
		case "down":
			if m.running || len(m.suggestions) == 0 {
				return m, nil
			}
			if m.selectedSug < len(m.suggestions)-1 {
				m.selectedSug++
			}
			return m, nil
		case "tab":
			if m.running {
				return m, nil
			}
			if len(m.suggestions) > 0 {
				c := slashCommands[m.suggestions[m.selectedSug]]
				m.input = "/" + c.Name
				m.updateSuggestions()
			}
			return m, nil
		case "backspace":
			if m.running {
				return m, nil
			}
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
				m.updateSuggestions()
			}
			return m, nil
		case "enter":
			if m.running {
				return m, nil
			}
			typed := strings.TrimSpace(m.input)
			if typed == "" {
				return m, nil
			}
			m.input = ""
			m.updateSuggestions()
			return m.handleSubmit(typed)
		default:
			if m.running {
				return m, nil
			}
			if len(msg.Runes) > 0 {
				m.input += string(msg.Runes)
				m.updateSuggestions()
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) withRuntimeDefaults() model {
	if m.runCLIFn == nil {
		m.runCLIFn = runCLICommandCmd
	}
	if m.runShellFn == nil {
		m.runShellFn = runShellCommandCmd
	}
	if m.lookPathFn == nil {
		m.lookPathFn = exec.LookPath
	}
	if m.nowFn == nil {
		m.nowFn = time.Now
	}
	return m
}

func (m model) handleSubmit(typed string) (tea.Model, tea.Cmd) {
	if strings.HasPrefix(typed, "/") {
		args := parseSlashCommand(typed)
		if len(args) == 0 {
			m.lastAction = "Unknown slash command."
			m.lastOutput = "Unknown command. Try /commands."
			return m, nil
		}
		if len(args) == 1 && args[0] == "__commands__" {
			m.lastAction = "Listed slash commands."
			m.lastOutput = formatCommandsHelp()
			return m, nil
		}
		if len(args) == 1 && args[0] == "__integrations__" {
			m.lastAction = "Checked integrations."
			m.lastOutput = formatIntegrations(m.status, m.lookPathFn)
			return m, nil
		}
		if len(args) >= 1 && args[0] == "__integrations__" {
			mode := "show"
			if len(args) > 1 {
				mode = strings.ToLower(strings.TrimSpace(args[1]))
			}
			switch mode {
			case "pin", "on":
				m.pinIntegrations = true
				m.lastAction = "Integrations pinned."
				m.lastOutput = "Integrations panel is now pinned in the workspace.\nUse /integrations unpin to disable."
				return m, nil
			case "unpin", "off":
				m.pinIntegrations = false
				m.lastAction = "Integrations unpinned."
				m.lastOutput = "Integrations panel is no longer pinned."
				return m, nil
			case "toggle":
				m.pinIntegrations = !m.pinIntegrations
				if m.pinIntegrations {
					m.lastAction = "Integrations pinned."
					m.lastOutput = "Integrations panel is now pinned in the workspace."
				} else {
					m.lastAction = "Integrations unpinned."
					m.lastOutput = "Integrations panel is no longer pinned."
				}
				return m, nil
			default:
				m.lastAction = "Checked integrations."
				m.lastOutput = formatIntegrations(m.status, m.lookPathFn)
				return m, nil
			}
		}
		if len(args) >= 1 && args[0] == "__provider__" {
			return m.handleProviderCommand(args[1:])
		}
		if len(args) >= 1 && args[0] == "__pipeline__" {
			return m.handlePipelineCommand(args[1:])
		}
		m.lastOutput = "hamunaptra " + strings.Join(args, " ")
		m.running = true
		m.lastAction = "Running: hamunaptra " + strings.Join(args, " ")
		return m, m.runCLIFn(args)
	}

	if strings.HasPrefix(typed, "!") {
		cmd, args := parseShellCommand(typed)
		if cmd == "" {
			m.lastAction = "Invalid shell command."
			m.lastOutput = "Invalid ! command. Example: !ls -la"
			return m, nil
		}
		if !shellAllowlist[cmd] {
			m.lastAction = "Blocked by shell allowlist."
			m.lastOutput = "Blocked: " + cmd + "\nAllowed: pwd, ls, git, cat, echo, whoami, date"
			return m, nil
		}
		full := append([]string{cmd}, args...)
		m.lastOutput = strings.Join(full, " ")
		m.running = true
		m.lastAction = "Running shell: " + strings.Join(full, " ")
		return m, m.runShellFn(cmd, args)
	}

	args := []string{"ask", typed}
	m.lastOutput = "hamunaptra ask …"
	m.running = true
	m.lastAction = "Running: hamunaptra ask ..."
	return m, m.runCLIFn(args)
}

func (m model) handleProviderCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) < 2 {
		m.lastAction = "Missing provider action."
		m.lastOutput = "Usage: /provider vercel setup|sync|report"
		return m, nil
	}
	provider := strings.ToLower(strings.TrimSpace(args[0]))
	action := strings.ToLower(strings.TrimSpace(args[1]))
	if provider != "vercel" {
		m.lastAction = "Unsupported provider."
		m.lastOutput = "Only vercel is supported in TUI flow for now.\nTry: /provider vercel setup"
		return m, nil
	}
	if !m.status.LoggedIn {
		m.lastAction = "Login required."
		m.lastOutput = "You are not logged in.\nRun /login first."
		return m, nil
	}
	if strings.TrimSpace(m.status.Project) == "" {
		m.lastAction = "Project link required."
		m.lastOutput = "No linked project found.\nRun /init first to link this folder."
		return m, nil
	}

	switch action {
	case "setup":
		runArgs := []string{"connect", "vercel"}
		m.lastOutput = "hamunaptra " + strings.Join(runArgs, " ")
		m.running = true
		m.lastAction = "Running: hamunaptra connect vercel"
		return m, m.runCLIFn(runArgs)
	case "sync":
		runArgs := []string{"sync"}
		m.lastOutput = "hamunaptra sync"
		m.running = true
		m.lastAction = "Running: hamunaptra sync"
		return m, m.runCLIFn(runArgs)
	case "report":
		runArgs := []string{"report", "--json"}
		m.postProcessMode = "report_json"
		m.lastOutput = "hamunaptra report --json"
		m.running = true
		m.lastAction = "Running: hamunaptra report --json"
		return m, m.runCLIFn(runArgs)
	default:
		m.lastAction = "Unknown provider action."
		m.lastOutput = "Use: /provider vercel setup|sync|report"
		return m, nil
	}
}

func (m model) handlePipelineCommand(args []string) (tea.Model, tea.Cmd) {
	provider := "vercel"
	if len(args) > 0 {
		provider = strings.ToLower(strings.TrimSpace(args[0]))
	}
	if provider != "vercel" {
		m.lastAction = "Unsupported pipeline provider."
		m.lastOutput = "Only /pipeline vercel is supported in TUI flow for now."
		return m, nil
	}
	m.lastAction = "Pipeline status (vercel)."
	m.lastOutput = formatPipelineGuidance(m.status, m.lookPathFn)
	return m, nil
}

func (m *model) updateSuggestions() {
	m.suggestions = m.suggestions[:0]
	m.selectedSug = 0
	q := strings.ToLower(strings.TrimSpace(m.input))
	if !strings.HasPrefix(q, "/") {
		return
	}
	q = strings.TrimPrefix(q, "/")
	for i, c := range slashCommands {
		if q == "" || strings.HasPrefix(c.Name, q) || strings.Contains(c.Name, q) {
			m.suggestions = append(m.suggestions, i)
		}
	}
}

func (m model) View() string {
	// Top hero panel.
	heroW := max(40, m.width-4)
	header := m.renderHeroPanel(heroW)

	// Bottom prompt area, anchored.
	inputText := m.input
	if strings.TrimSpace(inputText) == "" {
		inputText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render("type a command (/commands) or ask a question...")
	}
	prompt := promptStyle().
		Width(max(40, m.width-4)).
		Render("❯ " + m.renderInputLine(inputText))

	sugsText := m.renderSuggestions()
	bottomParts := []string{prompt}
	if strings.TrimSpace(sugsText) != "" {
		sugs := suggestionsStyle().
			Width(max(40, m.width-4)).
			Render(sugsText)
		bottomParts = append(bottomParts, sugs)
	}
	statusRow := renderStatusRow(max(40, m.width-4), "Now: "+m.lastAction+m.runningHint()+m.quitHint(), m.renderAuthFooter())
	bottomParts = append(bottomParts, statusStyle().Width(max(40, m.width-4)).Render(statusRow))
	bottom := lipgloss.JoinVertical(lipgloss.Left, bottomParts...)

	// Main workspace: single rolling surface — last output / activity, not a chat transcript.
	headerH := lipgloss.Height(header)
	available := m.height - headerH - lipgloss.Height(bottom) - 1
	mainHeight := available
	if mainHeight < 1 {
		mainHeight = 1
	}
	mainW := max(40, m.width-4)
	mainPanel := mainWorkspaceStyle().
		Width(mainW).
		Height(mainHeight).
		Render(m.renderMainWorkspace(max(1, mainHeight-2)))

	top := mainPanel

	// Keep breathing space above and below header.
	content := lipgloss.JoinVertical(lipgloss.Left, "", header, "", top)
	return placeWithBottom(content, bottom, m.width, m.height)
}

func (m model) renderHeroPanel(width int) string {
	if width < 40 {
		width = 40
	}
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}

	brand := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true).
		Render("Hamunaptra")
	ver := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Render("v" + buildVersion())
	title := " " + brand + " " + ver + " "
	top := "╭" + renderCutoutTopLine(innerW, title) + "╮"

	leftW := int(float64(innerW) * 0.54)
	if leftW < 28 {
		leftW = 28
	}
	if leftW > innerW-14 {
		leftW = innerW - 14
	}
	if leftW < 1 {
		leftW = innerW / 2
	}
	rightW := innerW - leftW - 3
	if rightW < 10 {
		rightW = 10
		leftW = innerW - rightW - 3
	}

	welcome := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB")).Render("Welcome back!")
	pyramidStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA"))
	pyramid := []string{
		pyramidStyle.Render("    █    "),
		pyramidStyle.Render("   ███   "),
		pyramidStyle.Render("  █████  "),
		pyramidStyle.Render(" ███████ "),
		pyramidStyle.Render("█████████"),
	}
	linksMuted := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	leftLines := []string{"", centerANSI(welcome, leftW), ""}
	for _, p := range pyramid {
		leftLines = append(leftLines, centerANSI(p, leftW))
	}
	leftLines = append(leftLines,
		"",
		centerANSI(linksMuted.Render("https://docs.hamunaptra.ai"), leftW),
		centerANSI(linksMuted.Render("https://github.com/rodrigobatini/hamunaptra-cli"), leftW),
	)

	gettingStarted := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A78BFA")).Render("Getting started")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	rightLines := []string{
		gettingStarted,
		"",
		muted.Render("Run /login to connect your account."),
		muted.Render("Use /integrations to check providers."),
		muted.Render("Type /commands to see everything."),
	}

	maxRows := len(leftLines)
	if len(rightLines) > maxRows {
		maxRows = len(rightLines)
	}

	body := make([]string, 0, maxRows)
	for i := 0; i < maxRows; i++ {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		row := "│" + padANSI(left, leftW) + " │ " + padANSI(right, rightW) + "│"
		body = append(body, row)
	}

	bottom := "╰" + strings.Repeat("─", innerW) + "╯"
	return strings.Join(append([]string{top}, append(body, bottom)...), "\n")
}

func renderCutoutTopLine(width int, title string) string {
	tw := lipgloss.Width(title)
	if tw >= width {
		return strings.Repeat("─", width)
	}
	left := 2
	if left > width-tw {
		left = 0
	}
	right := width - tw - left
	if right < 0 {
		right = 0
	}
	return strings.Repeat("─", left) + title + strings.Repeat("─", right)
}

func padANSI(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return truncateWidth(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

func centerANSI(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return truncateWidth(s, width)
	}
	left := (width - w) / 2
	right := width - w - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func runCLICommandCmd(args []string) tea.Cmd {
	return func() tea.Msg {
		exe, err := os.Executable()
		if err != nil {
			return runResultMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, exe, args...)
		out, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return runResultMsg{output: string(out), err: fmt.Errorf("command timed out")}
		}
		return runResultMsg{output: string(out), err: err}
	}
}

func runShellCommandCmd(bin string, args []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			return runResultMsg{output: string(out), err: fmt.Errorf("shell command timed out")}
		}
		return runResultMsg{output: string(out), err: err}
	}
}

func parseSlashCommand(input string) []string {
	raw := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	if raw == "" {
		return nil
	}
	parts := tokenize(raw)
	if len(parts) == 0 {
		return nil
	}
	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "commands":
		return []string{"__commands__"}
	case "integrations":
		out := []string{"__integrations__"}
		if len(parts) > 1 {
			out = append(out, parts[1:]...)
		}
		return out
	case "provider":
		out := []string{"__provider__"}
		if len(parts) > 1 {
			out = append(out, parts[1:]...)
		}
		return out
	case "pipeline":
		out := []string{"__pipeline__"}
		if len(parts) > 1 {
			out = append(out, parts[1:]...)
		}
		return out
	case "login", "sync", "report", "anomalies":
		return append([]string{cmd}, parts[1:]...)
	case "help":
		return []string{"--help"}
	case "init", "connect", "ask":
		return append([]string{cmd}, parts[1:]...)
	default:
		return nil
	}
}

func parseShellCommand(input string) (string, []string) {
	raw := strings.TrimSpace(strings.TrimPrefix(input, "!"))
	if raw == "" {
		return "", nil
	}
	parts := tokenize(raw)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}

func tokenize(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			if ch == quote {
				inQuote = false
				continue
			}
			cur.WriteByte(ch)
			continue
		}
		if ch == '\'' || ch == '"' {
			inQuote = true
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

func formatCommandsHelp() string {
	var lines []string
	lines = append(lines, "Slash commands")
	lines = append(lines, "")
	for _, c := range slashCommands {
		lines = append(lines, fmt.Sprintf("  /%s  — %s", c.Name, c.Desc))
	}
	lines = append(lines, "")
	lines = append(lines, "Integrations panel modes")
	lines = append(lines, "  /integrations         show once")
	lines = append(lines, "  /integrations pin     keep visible")
	lines = append(lines, "  /integrations unpin   disable pinned panel")
	lines = append(lines, "  /integrations toggle  switch pinned mode")
	lines = append(lines, "")
	lines = append(lines, "Vercel real pipeline")
	lines = append(lines, "  /provider vercel setup   run connect vercel")
	lines = append(lines, "  /provider vercel sync    run sync")
	lines = append(lines, "  /provider vercel report  run report --json")
	lines = append(lines, "  /pipeline vercel         show readiness + next step")
	lines = append(lines, "")
	lines = append(lines, "Shell: !<command> (allowlist)")
	return strings.Join(lines, "\n")
}

func formatIntegrations(status statusData, lookPath func(string) (string, error)) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("●")
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("●")
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Render("●")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#94A3B8"))
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5E7EB")).Render("Integrations Status")
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	authState := red + " not logged in"
	if status.LoggedIn {
		authState = green + " logged in"
	}
	projectState := yellow + " not linked"
	if strings.TrimSpace(status.Project) != "" {
		projectState = green + " linked (" + status.Project + ")"
	}
	connectionState := yellow + " not connected"
	if status.VercelConnected {
		connectionState = green + " connected"
		if status.VercelSource != "" {
			connectionState += " (" + status.VercelSource + ")"
		}
	}
	lastSyncState := yellow + " never"
	if strings.TrimSpace(status.VercelLastSync) != "" {
		lastSyncState = green + " " + status.VercelLastSync
	}
	lastErrorState := green + " none"
	if strings.TrimSpace(status.VercelLastError) != "" {
		lastErrorState = red + " " + status.VercelLastError
	}

	toolState := func(bin string) string {
		if _, err := lookPath(bin); err != nil {
			return red + " missing"
		}
		return green + " available"
	}
	nextStep := "run /provider vercel setup"
	if !status.LoggedIn {
		nextStep = "run /login"
	} else if strings.TrimSpace(status.Project) == "" {
		nextStep = "run /init"
	} else if !status.VercelConnected {
		nextStep = "run /provider vercel setup"
	} else if strings.TrimSpace(status.VercelLastSync) == "" {
		nextStep = "run /provider vercel sync"
	} else {
		nextStep = "run /provider vercel report"
	}

	body := strings.Join([]string{
		title,
		"",
		fmt.Sprintf("  auth            %s", authState),
		fmt.Sprintf("  project         %s", projectState),
		fmt.Sprintf("  vercel conn     %s", connectionState),
		fmt.Sprintf("  vercel sync     %s", lastSyncState),
		fmt.Sprintf("  vercel error    %s", lastErrorState),
		"",
		fmt.Sprintf("  vercel cli      %s", toolState("vercel")),
		fmt.Sprintf("  supabase cli    %s", toolState("supabase")),
		fmt.Sprintf("  neon cli        %s", toolState("neonctl")),
		fmt.Sprintf("  render cli      %s", toolState("render")),
		"",
		muted.Render("Next step: " + nextStep),
		"",
		muted.Render("Legend:"),
		muted.Render(fmt.Sprintf("  %s ready   %s missing   %s attention", green, red, yellow)),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Render(body)
}

func formatPipelineGuidance(status statusData, lookPath func(string) (string, error)) string {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("OK")
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("#EAB308")).Render("TODO")
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	auth := yellow + " login required"
	if status.LoggedIn {
		auth = green + " authenticated"
	}
	project := yellow + " project not linked"
	if strings.TrimSpace(status.Project) != "" {
		project = green + " linked (" + status.Project + ")"
	}
	conn := yellow + " connection missing"
	if status.VercelConnected {
		conn = green + " connected"
	}
	lastSync := yellow + " no sync yet"
	if strings.TrimSpace(status.VercelLastSync) != "" {
		lastSync = green + " last sync " + status.VercelLastSync
	}
	lastErr := green + " none"
	if strings.TrimSpace(status.VercelLastError) != "" {
		lastErr = yellow + " " + status.VercelLastError
	}
	vercelCLI := yellow + " vercel cli missing (optional)"
	if _, err := lookPath("vercel"); err == nil {
		vercelCLI = green + " vercel cli available"
	}

	next := "/login"
	if status.LoggedIn && strings.TrimSpace(status.Project) == "" {
		next = "/init"
	}
	if status.LoggedIn && strings.TrimSpace(status.Project) != "" && !status.VercelConnected {
		next = "/provider vercel setup"
	}
	if status.LoggedIn && strings.TrimSpace(status.Project) != "" && status.VercelConnected && strings.TrimSpace(status.VercelLastSync) == "" {
		next = "/provider vercel sync"
	}
	if status.LoggedIn && strings.TrimSpace(status.Project) != "" && status.VercelConnected && strings.TrimSpace(status.VercelLastSync) != "" {
		next = "/provider vercel report"
	}

	return strings.Join([]string{
		"Vercel pipeline readiness",
		"",
		"  auth:       " + auth,
		"  project:    " + project,
		"  connection: " + conn,
		"  sync:       " + lastSync,
		"  last error: " + lastErr,
		"  vercel cli: " + vercelCLI,
		"",
		"Recommended next step:",
		"  " + next,
		"",
		"After setup:",
		"  /provider vercel sync",
		"  /provider vercel report",
	}, "\n")
}

func humanizeRunError(errText string) string {
	msg := strings.TrimSpace(errText)
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "not logged in"):
		return "not logged in. Run /login."
	case strings.Contains(lower, "no project linked"), strings.Contains(lower, "project_id not found"):
		return "no linked project. Run /init first."
	case strings.Contains(lower, "timed out"), strings.Contains(lower, "timeout"):
		return "request timed out. Check network/API and retry."
	case strings.Contains(lower, "connect:"), strings.Contains(lower, "sync:"), strings.Contains(lower, "report"):
		return "backend/provider request failed: " + msg
	default:
		return msg
	}
}

func summarizeReportOutput(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "(empty report output)"
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return trimmed
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return trimmed
	}

	total, okTotal := pickNumber(obj, "total", "total_cost", "amount", "cost")
	currency, _ := pickString(obj, "currency")
	from, _ := pickString(obj, "from", "start", "period_start")
	to, _ := pickString(obj, "to", "end", "period_end")
	servicesCount := pickArrayLen(obj, "services", "breakdown", "items")

	lines := []string{"Cost report summary"}
	if okTotal {
		totalLine := fmt.Sprintf("  total: %.2f", total)
		if currency != "" {
			totalLine += " " + strings.ToUpper(currency)
		}
		lines = append(lines, totalLine)
	}
	if from != "" || to != "" {
		lines = append(lines, fmt.Sprintf("  period: %s -> %s", fallbackString(from, "?"), fallbackString(to, "?")))
	}
	if servicesCount > 0 {
		lines = append(lines, fmt.Sprintf("  items: %d", servicesCount))
	}
	if len(lines) == 1 {
		return trimmed
	}
	lines = append(lines, "", "Raw JSON", shortMultiline(trimmed, 12))
	return strings.Join(lines, "\n")
}

func pickNumber(obj map[string]any, keys ...string) (float64, bool) {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			switch n := v.(type) {
			case float64:
				return n, true
			case int:
				return float64(n), true
			}
		}
	}
	return 0, false
}

func pickString(obj map[string]any, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s, true
			}
		}
	}
	return "", false
}

func pickArrayLen(obj map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := obj[k]; ok {
			if arr, ok := v.([]any); ok {
				return len(arr)
			}
		}
	}
	return 0
}

func fallbackString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func (m model) renderSuggestions() string {
	trimmed := strings.TrimSpace(m.input)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "!") {
		return "Shell mode\n\n!<command> [args]\nAllowed: pwd, ls, git, cat, echo, whoami, date"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	if len(m.suggestions) == 0 {
		return "Slash commands\n\nNo match. Use ↑/↓ and Tab after typing /."
	}
	const maxVisible = 6
	start := m.selectedSug - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.suggestions) {
		end = len(m.suggestions)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}
	var lines []string
	lines = append(lines, "Slash suggestions")
	lines = append(lines, "")
	for i := start; i < end; i++ {
		idx := m.suggestions[i]
		c := slashCommands[idx]
		prefix := "  "
		if i == m.selectedSug {
			prefix = "▸ "
		}
		line := fmt.Sprintf("%s/%s", prefix, c.Name)
		if i == m.selectedSug {
			line = lipgloss.NewStyle().Foreground(lipgloss.Color("#30D0CF")).Bold(true).Render(line)
		}
		desc := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B95A7")).Render(" — " + c.Desc)
		lines = append(lines, line+desc)
	}
	if len(m.suggestions) > maxVisible {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.suggestions)))
	}
	return strings.Join(lines, "\n")
}

func (m model) renderMainWorkspace(maxLines int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	out := strings.TrimSpace(m.lastOutput)
	if m.pinIntegrations {
		panel := formatIntegrations(m.status, m.lookPathFn)
		if out == "" {
			out = panel
		} else {
			out = panel + "\n\n" + "----" + "\n" + out
		}
	}
	if out == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B")).Render("—")
	}
	lines := strings.Split(out, "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], "… (truncated)")
	}
	return strings.Join(lines, "\n")
}

func (m model) renderBranding(maxW int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)

	candidates := []string{
		hamunaptraFilledTiny(),
		hamunaptraFilledSmall(),
		"HAMUNAPTRA",
	}
	for _, raw := range candidates {
		if lipgloss.Width(raw) <= maxW {
			return style.Render(raw)
		}
	}
	return style.Render("HAMUNAPTRA")
}

func hamunaptraFilledTiny() string {
	return strings.Join([]string{
		"████ HAMUNAPTRA ████",
		"████████████████████",
	}, "\n")
}

func hamunaptraFilledSmall() string {
	return strings.Join([]string{
		"██╗  ██╗ █████╗ ███╗   ███╗██╗   ██╗",
		"███████║███████║██╔████╔██║██║   ██║",
		"██║  ██║██║  ██║██║ ╚═╝ ██║╚██████╔╝",
	}, "\n")
}

func (m model) renderAuthFooter() string {
	pinHint := ""
	if m.pinIntegrations {
		pinHint = "  |  Integrations: pinned"
	}
	if m.status.LoggedIn {
		return "Auth: logged in" + pinHint
	}
	return "Auth: not logged in  |  use /login" + pinHint
}

func (m model) renderHeaderInfo() string {
	return strings.Join([]string{
		"Info",
		fmt.Sprintf("  version: %s", buildVersion()),
		"  docs:    https://docs.hamunaptra.ai",
		"  github:  https://github.com/rodrigobatini/hamunaptra-cli",
	}, "\n")
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && info != nil {
		v := strings.TrimSpace(info.Main.Version)
		if v != "" && v != "(devel)" {
			return v
		}
	}
	return "0.1.0-dev"
}

func (m model) runningHint() string {
	if m.running {
		return "  [running...]"
	}
	return ""
}

func (m model) quitHint() string {
	if time.Now().Before(m.quitArmedUntil) {
		return "  [Ctrl+C again to quit]"
	}
	return ""
}

func collectStatus() statusData {
	out := statusData{}
	if home, err := os.UserHomeDir(); err == nil {
		out.ConfigPath = filepath.Join(home, ".config", "hamunaptra", "config.json")
	}
	cfg, err := configfile.Load()
	if err == nil && cfg != nil {
		out.LoggedIn = strings.TrimSpace(cfg.AccessToken) != ""
		out.APIBase = cfg.APIBase
		if out.APIBase == "" {
			out.APIBase = configfile.DefaultAPIBase()
		}
	}
	wd, err := os.Getwd()
	if err == nil {
		out.WorkDir = wd
		if pid, err := localproj.ReadID(wd); err == nil {
			out.Project = pid
		}
	}
	if err == nil && cfg != nil && out.LoggedIn {
		out = fillRemoteConnectionStatus(out, cfg.AccessToken)
	}
	return out
}

func fillRemoteConnectionStatus(out statusData, accessToken string) statusData {
	if strings.TrimSpace(accessToken) == "" || strings.TrimSpace(out.Project) == "" {
		return out
	}
	c := api.New(out.APIBase, accessToken)
	c.Client.Timeout = 4 * time.Second
	conns, err := c.ListConnections(out.Project)
	if err != nil {
		return out
	}
	for _, conn := range conns {
		if strings.ToLower(strings.TrimSpace(conn.Provider)) != "vercel" {
			continue
		}
		out.VercelConnected = true
		out.VercelSource = conn.Source
		if conn.LastSyncAt != nil {
			out.VercelLastSync = strings.TrimSpace(*conn.LastSyncAt)
		}
		if conn.LastError != nil {
			out.VercelLastError = strings.TrimSpace(*conn.LastError)
		}
		break
	}
	return out
}

func placeWithBottom(top, bottom string, w, h int) string {
	if w <= 0 || h <= 0 {
		return top + "\n" + bottom
	}
	topLines := strings.Split(top, "\n")
	bottomLines := strings.Split(bottom, "\n")

	// Keep bottom always visible; trim top first when height is tight.
	maxTop := h - len(bottomLines)
	if maxTop < 0 {
		if len(bottomLines) > h {
			bottomLines = bottomLines[len(bottomLines)-h:]
		}
		return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, strings.Join(bottomLines, "\n"))
	}
	if len(topLines) > maxTop {
		topLines = topLines[:maxTop]
	}
	gap := h - len(topLines) - len(bottomLines)
	if gap < 0 {
		gap = 0
	}
	parts := []string{}
	if len(topLines) > 0 {
		parts = append(parts, strings.Join(topLines, "\n"))
	}
	if gap > 0 {
		parts = append(parts, strings.Repeat("\n", gap))
	}
	if len(bottomLines) > 0 {
		parts = append(parts, strings.Join(bottomLines, "\n"))
	}
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, strings.Join(parts, "\n"))
}

func asciiLogo() string {
	return strings.Join([]string{
		"██╗  ██╗ █████╗ ███╗   ███╗██╗   ██╗███╗   ██╗ █████╗ ██████╗ ████████╗██████╗  █████╗",
		"██║  ██║██╔══██╗████╗ ████║██║   ██║████╗  ██║██╔══██╗██╔══██╗╚══██╔══╝██╔══██╗██╔══██╗",
		"███████║███████║██╔████╔██║██║   ██║██╔██╗ ██║███████║██████╔╝   ██║   ██████╔╝███████║",
		"██╔══██║██╔══██║██║╚██╔╝██║██║   ██║██║╚██╗██║██╔══██║██╔═══╝    ██║   ██╔══██╗██╔══██║",
		"██║  ██║██║  ██║██║ ╚═╝ ██║╚██████╔╝██║ ╚████║██║  ██║██║        ██║   ██║  ██║██║  ██║",
		"╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝        ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝",
	}, "\n")
}

func asciiLogoSmall() string {
	return strings.Join([]string{
		"██╗  ██╗ █████╗ ███╗   ███╗██╗   ██╗███╗   ██╗ █████╗ ██████╗ ████████╗██████╗  █████╗",
		"███████║███████║██╔████╔██║██║   ██║██╔██╗ ██║███████║██████╔╝   ██║   ██████╔╝███████║",
	}, "\n")
}

func asciiLogoTiny() string {
	return strings.Join([]string{
		"██╗  ██╗ █████╗ ███╗   ███╗██╗   ██╗",
		"███████║███████║██╔████╔██║██║   ██║",
	}, "\n")
}

func boxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3A3F4B")).
		Padding(0, 1)
}

func mainWorkspaceStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#334155")).
		Padding(0, 1)
}

func promptStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#8B5CF6")).
		Padding(0, 1)
}

func (m model) renderInputLine(inputText string) string {
	cursor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111827")).
		Background(lipgloss.Color("#FFFFFF")).
		Render(" ")
	if strings.TrimSpace(m.input) == "" {
		return cursor + inputText
	}
	return inputText + cursor
}

func suggestionsStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(0, 1)
}

func statusStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#94A3B8"))
}

func statusRightStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#94A3B8")).
		Align(lipgloss.Right)
}

func renderStatusRow(width int, left, right string) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if rw >= width {
		return truncateWidth(right, width)
	}
	if lw+rw+2 > width {
		left = truncateWidth(left, width-rw-2)
		lw = lipgloss.Width(left)
	}
	gap := width - lw - rw
	if gap < 2 {
		gap = 2
	}
	return left + strings.Repeat(" ", gap) + right
}

func truncateWidth(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW == 1 {
		return "…"
	}
	target := maxW - 1
	runes := []rune(s)
	var b strings.Builder
	for _, r := range runes {
		next := b.String() + string(r)
		if lipgloss.Width(next) > target {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}

func headerInfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Padding(0, 0)
}

func headerBoxStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#8B5CF6")).
		Padding(0, 1)
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func short(v string, n int) string {
	if len(v) <= n {
		return v
	}
	if n < 4 {
		return v[:n]
	}
	return v[:n-3] + "..."
}

func shortMultiline(s string, maxLines int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], "... (truncated)")
	}
	return strings.Join(lines, " ")
}

func pill(txt string, ok bool) string {
	if ok {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(lipgloss.Color("#34D399")).
			Padding(0, 1).
			Render(txt)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Background(lipgloss.Color("#374151")).
		Padding(0, 1).
		Render(txt)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
