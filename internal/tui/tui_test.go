package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseSlashCommand(t *testing.T) {
	got := parseSlashCommand("/commands")
	if len(got) != 1 || got[0] != "__commands__" {
		t.Fatalf("unexpected /commands parse: %#v", got)
	}

	got = parseSlashCommand("/login")
	if len(got) != 1 || got[0] != "login" {
		t.Fatalf("unexpected /login parse: %#v", got)
	}

	got = parseSlashCommand("/integrations pin")
	if len(got) != 2 || got[0] != "__integrations__" || got[1] != "pin" {
		t.Fatalf("unexpected /integrations pin parse: %#v", got)
	}
	got = parseSlashCommand("/provider vercel report")
	if len(got) != 3 || got[0] != "__provider__" || got[1] != "vercel" || got[2] != "report" {
		t.Fatalf("unexpected /provider vercel report parse: %#v", got)
	}
	got = parseSlashCommand("/pipeline vercel")
	if len(got) != 2 || got[0] != "__pipeline__" || got[1] != "vercel" {
		t.Fatalf("unexpected /pipeline vercel parse: %#v", got)
	}

	got = parseSlashCommand("/unknown")
	if got != nil {
		t.Fatalf("expected nil for unknown command, got: %#v", got)
	}
}

func TestParseShellCommandAndTokenize(t *testing.T) {
	bin, args := parseShellCommand(`!echo "hello world"`)
	if bin != "echo" {
		t.Fatalf("expected echo bin, got %q", bin)
	}
	if len(args) != 1 || args[0] != "hello world" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestUpdateSuggestions(t *testing.T) {
	m := model{input: "/in"}
	m.updateSuggestions()
	if len(m.suggestions) == 0 {
		t.Fatal("expected suggestions for /in")
	}
	foundIntegrations := false
	for _, idx := range m.suggestions {
		if slashCommands[idx].Name == "integrations" {
			foundIntegrations = true
			break
		}
	}
	if !foundIntegrations {
		t.Fatal("expected integrations to be suggested")
	}
}

func TestHandleSubmitCommandsAndRunningTransition(t *testing.T) {
	m := model{
		runCLIFn: func(args []string) tea.Cmd {
			return func() tea.Msg {
				return runResultMsg{output: "ok", err: nil}
			}
		},
	}

	updated, cmd := m.handleSubmit("/commands")
	um := updated.(model)
	if !strings.Contains(um.lastOutput, "Slash commands") {
		t.Fatalf("expected slash commands output, got: %q", um.lastOutput)
	}
	if cmd != nil {
		t.Fatal("expected no async cmd for /commands")
	}

	updated, cmd = m.handleSubmit("/login")
	um = updated.(model)
	if !um.running {
		t.Fatal("expected running state for /login")
	}
	if cmd == nil {
		t.Fatal("expected async cmd for /login")
	}

	msg := cmd()
	updatedModel, _ := um.Update(msg)
	um = updatedModel.(model)
	if um.running {
		t.Fatal("expected running=false after runResultMsg")
	}
	if um.lastAction != "Done." {
		t.Fatalf("expected Done. action, got %q", um.lastAction)
	}
}

func TestHandleSubmitShellAllowlist(t *testing.T) {
	m := model{}
	updated, cmd := m.handleSubmit("!rm -rf /")
	um := updated.(model)
	if cmd != nil {
		t.Fatal("expected no cmd for blocked shell command")
	}
	if !strings.Contains(um.lastOutput, "Blocked") {
		t.Fatalf("expected blocked message, got %q", um.lastOutput)
	}
}

func TestFormatIntegrationsUsesRealStatusSignals(t *testing.T) {
	status := statusData{
		LoggedIn: true,
		Project:  "proj_123",
	}
	lookPath := func(file string) (string, error) {
		if file == "vercel" || file == "supabase" {
			return "/usr/bin/" + file, nil
		}
		return "", errors.New("missing")
	}

	out := formatIntegrations(status, lookPath)
	plain := stripANSI(out)
	if !strings.Contains(plain, "auth") || !strings.Contains(plain, "logged in") {
		t.Fatalf("expected logged in state in integrations output: %q", plain)
	}
	if !strings.Contains(plain, "linked (proj_123)") {
		t.Fatalf("expected project linkage in integrations output: %q", plain)
	}
	if !strings.Contains(plain, "vercel cli") || !strings.Contains(plain, "available") {
		t.Fatalf("expected vercel available in integrations output: %q", plain)
	}
	if !strings.Contains(plain, "neon cli") || !strings.Contains(plain, "missing") {
		t.Fatalf("expected neon missing in integrations output: %q", plain)
	}
	if !strings.Contains(plain, "Next step") {
		t.Fatalf("expected next step hint in integrations output: %q", plain)
	}
}

func TestIntegrationsPinToggleFlow(t *testing.T) {
	m := model{
		lookPathFn: func(file string) (string, error) { return "", errors.New("missing") },
	}

	updated, _ := m.handleSubmit("/integrations pin")
	um := updated.(model)
	if !um.pinIntegrations {
		t.Fatal("expected pinIntegrations=true after pin")
	}
	main := um.renderMainWorkspace(200)
	if !strings.Contains(stripANSI(main), "Integrations Status") {
		t.Fatalf("expected pinned integrations in main workspace: %q", stripANSI(main))
	}

	updated, _ = um.handleSubmit("/integrations unpin")
	um = updated.(model)
	if um.pinIntegrations {
		t.Fatal("expected pinIntegrations=false after unpin")
	}
}

func TestAuthFooterShowsPinnedIndicator(t *testing.T) {
	m := model{
		pinIntegrations: true,
		status: statusData{
			LoggedIn: true,
		},
	}
	footer := m.renderAuthFooter()
	if !strings.Contains(footer, "Integrations: pinned") {
		t.Fatalf("expected pinned indicator in auth footer, got %q", footer)
	}
}

func TestProviderVercelReportUsesPostProcessing(t *testing.T) {
	m := model{
		status: statusData{LoggedIn: true, Project: "proj_1"},
		runCLIFn: func(args []string) tea.Cmd {
			if len(args) != 2 || args[0] != "report" || args[1] != "--json" {
				t.Fatalf("unexpected args: %#v", args)
			}
			return func() tea.Msg {
				return runResultMsg{
					output: `{"total":123.45,"currency":"usd","from":"2026-03-01","to":"2026-03-31","services":[{"name":"vercel"}]}`,
				}
			}
		},
	}
	updated, cmd := m.handleSubmit("/provider vercel report")
	um := updated.(model)
	if !um.running {
		t.Fatal("expected running true for provider report")
	}
	if um.postProcessMode != "report_json" {
		t.Fatalf("expected report_json post process mode, got %q", um.postProcessMode)
	}
	msg := cmd()
	updatedModel, _ := um.Update(msg)
	um = updatedModel.(model)
	plain := stripANSI(um.lastOutput)
	if !strings.Contains(plain, "Cost report summary") {
		t.Fatalf("expected summarized report output, got: %q", plain)
	}
	if !strings.Contains(plain, "total: 123.45 USD") {
		t.Fatalf("expected total in summary, got: %q", plain)
	}
}

func TestProviderVercelPrechecks(t *testing.T) {
	m := model{
		status: statusData{LoggedIn: false},
	}
	updated, cmd := m.handleSubmit("/provider vercel setup")
	if cmd != nil {
		t.Fatal("expected no command when not logged in")
	}
	um := updated.(model)
	if !strings.Contains(um.lastOutput, "/login") {
		t.Fatalf("expected login guidance, got: %q", um.lastOutput)
	}

	m = model{
		status: statusData{LoggedIn: true, Project: ""},
	}
	updated, cmd = m.handleSubmit("/provider vercel setup")
	if cmd != nil {
		t.Fatal("expected no command when project missing")
	}
	um = updated.(model)
	if !strings.Contains(um.lastOutput, "/init") {
		t.Fatalf("expected init guidance, got: %q", um.lastOutput)
	}
}

func TestPipelineVercelGuidance(t *testing.T) {
	m := model{
		status: statusData{LoggedIn: true, Project: "proj_x"},
		lookPathFn: func(file string) (string, error) {
			if file == "vercel" {
				return "/usr/bin/vercel", nil
			}
			return "", errors.New("missing")
		},
	}
	updated, cmd := m.handleSubmit("/pipeline vercel")
	if cmd != nil {
		t.Fatal("expected no async cmd for pipeline guidance")
	}
	um := updated.(model)
	plain := stripANSI(um.lastOutput)
	if !strings.Contains(plain, "Vercel pipeline readiness") {
		t.Fatalf("expected pipeline title, got: %q", plain)
	}
	if !strings.Contains(plain, "/provider vercel setup") {
		t.Fatalf("expected next-step recommendation, got: %q", plain)
	}
}

func TestCtrlCArmingFlow(t *testing.T) {
	now := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	m := model{nowFn: func() time.Time { return now }}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Fatal("first ctrl+c should not quit immediately")
	}
	um := updated.(model)
	if !strings.Contains(um.lastOutput, "Press Ctrl+C again to quit") {
		t.Fatalf("unexpected first ctrl+c output: %q", um.lastOutput)
	}

	updated, cmd = um.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("second ctrl+c should trigger quit cmd")
	}
}

func TestRenderStatusRowAlignmentAndTruncation(t *testing.T) {
	row := renderStatusRow(40, "Now: "+strings.Repeat("x", 80), "Auth: logged in")
	plain := stripANSI(row)
	if !strings.Contains(plain, "Auth: logged in") {
		t.Fatalf("expected auth in status row: %q", plain)
	}
	if len([]rune(plain)) > 60 {
		t.Fatalf("expected truncation to keep row bounded, got len=%d", len([]rune(plain)))
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEsc {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEsc = false
			}
			continue
		}
		if ch == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
