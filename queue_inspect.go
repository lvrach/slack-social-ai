package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lvrach/slack-social-ai/internal/config"
	"github.com/lvrach/slack-social-ai/internal/history"
	"github.com/lvrach/slack-social-ai/internal/schedule"
)

// QueueInspectCmd opens an interactive TUI to browse and delete queue items.
type QueueInspectCmd struct{}

func (cmd *QueueInspectCmd) Run(globals *Globals) error {
	// JSON mode: fall back to queue show --json (TUI not meaningful for scripts).
	if globals.JSON {
		return (&QueueShowCmd{}).Run(globals)
	}

	entries, err := history.Queued()
	if err != nil {
		return newCLIError(ExitRuntimeError, "load_queue",
			fmt.Sprintf("Failed to load queue: %s", err))
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "Queue is empty.")
		return nil
	}

	cfg, _ := config.Load()
	lastPublished, _ := history.LastPublishedTime()
	now := time.Now().UTC()

	predictions := schedule.PredictPublishTimes(entries, cfg.Schedule, lastPublished, now)

	m := newInspectModel(predictions)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("inspect TUI: %w", err)
	}

	fm := finalModel.(inspectModel)
	if fm.deleted > 0 {
		fmt.Fprintf(os.Stdout, "Removed %d item(s) from queue.\n", fm.deleted)
	}
	return nil
}

const (
	inspectLeftPaneWidth = 26 // width of the list pane
	inspectSepWidth      = 3  // " │ " separator between panes
	minSplitWidth        = 60 // minimum terminal width for horizontal split
)

// inspectModel is the Bubble Tea model for the queue inspector.
type inspectModel struct {
	predictions     []schedule.Prediction
	renderedContent []string // pre-cached glamour output per item
	cursor          int
	deleted         int
	width, height   int
	message         string // transient status message
	detailViewport  viewport.Model
	focusDetail     bool
	confirmDelete   bool
	listOffset      int
}

func newInspectModel(predictions []schedule.Prediction) inspectModel {
	vp := viewport.New(80, 10)
	// Remove "d" from half-page-down (conflicts with delete key).
	vp.KeyMap.HalfPageDown = key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	return inspectModel{
		predictions:    predictions,
		detailViewport: vp,
	}
}

func (m inspectModel) Init() tea.Cmd {
	return nil
}

func (m inspectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 1. Delete confirmation takes priority over everything.
		if m.confirmDelete {
			switch msg.String() {
			case "y":
				return m.doDelete()
			default:
				m.confirmDelete = false
			}
			return m, nil
		}

		// 2. Global keys.
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			if m.width >= minSplitWidth && len(m.predictions) > 0 {
				m.focusDetail = !m.focusDetail
			}
			return m, nil

		case "d", "backspace", "delete":
			if !m.focusDetail && len(m.predictions) > 0 {
				m.confirmDelete = true
			}
			return m, nil
		}

		// 3. Route to focused pane (viewport handles its own keys).
		if m.focusDetail {
			var cmd tea.Cmd
			m.detailViewport, cmd = m.detailViewport.Update(msg)
			return m, cmd
		}

		// 4. List navigation.
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.message = ""
				m.syncDetailContent()
				m.syncListScroll()
			}
		case "down", "j":
			if m.cursor < len(m.predictions)-1 {
				m.cursor++
				m.message = ""
				m.syncDetailContent()
				m.syncListScroll()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.renderAllContent()
		m.updateViewportSize()
		m.syncDetailContent()
		m.syncListScroll()
	}

	return m, nil
}

// doDelete removes the currently selected entry and adjusts model state.
func (m inspectModel) doDelete() (tea.Model, tea.Cmd) {
	m.confirmDelete = false
	if m.cursor >= len(m.predictions) {
		return m, nil
	}

	entry := m.predictions[m.cursor].Entry
	found, err := history.Remove(entry.ID)
	if err != nil || !found {
		return m, nil
	}

	m.predictions = append(m.predictions[:m.cursor], m.predictions[m.cursor+1:]...)
	if m.renderedContent != nil {
		m.renderedContent = append(m.renderedContent[:m.cursor], m.renderedContent[m.cursor+1:]...)
	}
	for i := range m.predictions {
		m.predictions[i].Position = i + 1
	}
	m.deleted++
	m.message = fmt.Sprintf("Deleted: %s", truncate(firstLine(entry.Message), 40))

	if len(m.predictions) == 0 {
		return m, tea.Quit
	}
	if m.cursor >= len(m.predictions) {
		m.cursor = len(m.predictions) - 1
	}
	m.syncDetailContent()
	m.syncListScroll()
	return m, nil
}

// contentRows returns the number of rows available for the content area.
func (m inspectModel) contentRows() int {
	overhead := 2 // title + help
	if m.width >= minSplitWidth {
		overhead += 2 // top border + bottom border
	}
	if m.message != "" {
		overhead++
	}
	return max(m.height-overhead, 1)
}

// rightPaneWidth returns the width available for the detail pane.
func (m inspectModel) rightPaneWidth() int {
	return max(m.width-inspectLeftPaneWidth-inspectSepWidth, 1)
}

// renderAllContent pre-renders all messages via glamour for the detail pane.
func (m *inspectModel) renderAllContent() {
	if m.width < minSplitWidth {
		m.renderedContent = nil
		return
	}
	rightW := m.rightPaneWidth()
	m.renderedContent = make([]string, len(m.predictions))
	for i, p := range m.predictions {
		m.renderedContent[i] = renderMrkdwn(p.Entry.Message, max(rightW-2, 20))
	}
}

// updateViewportSize recalculates the detail viewport dimensions.
func (m *inspectModel) updateViewportSize() {
	if m.width < minSplitWidth {
		return
	}
	rows := m.contentRows()
	vpHeight := max(rows-2, 1) // subtract header + divider in right pane
	m.detailViewport.Width = m.rightPaneWidth()
	m.detailViewport.Height = vpHeight
}

// syncDetailContent sets the viewport to the currently selected item's content.
func (m *inspectModel) syncDetailContent() {
	if len(m.renderedContent) == 0 || m.cursor >= len(m.renderedContent) {
		m.detailViewport.SetContent("")
		return
	}
	m.detailViewport.SetContent(m.renderedContent[m.cursor])
	m.detailViewport.GotoTop()
}

// syncListScroll ensures the cursor is visible within the list pane.
func (m *inspectModel) syncListScroll() {
	rows := m.contentRows()
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	}
	if m.cursor >= m.listOffset+rows {
		m.listOffset = m.cursor - rows + 1
	}
}

// --- View styles ---

var (
	inspectTitleStyle = lipgloss.NewStyle().Bold(true)
	inspectDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	inspectHelpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	inspectMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func (m inspectModel) View() string {
	var b strings.Builder

	// Title.
	b.WriteString(inspectTitleStyle.Render(
		fmt.Sprintf("Queue (%d messages)", len(m.predictions))))
	b.WriteString("\n")

	if len(m.predictions) == 0 {
		b.WriteString(inspectHelpStyle.Render("q: quit"))
		return b.String()
	}

	if m.width < minSplitWidth {
		m.viewNarrow(&b)
	} else {
		m.viewSplit(&b)
	}

	// Transient status message.
	if m.message != "" {
		b.WriteString(inspectMsgStyle.Render(m.message))
		b.WriteString("\n")
	}

	// Help bar.
	b.WriteString(inspectHelpStyle.Render(m.helpText()))

	return b.String()
}

// viewNarrow renders a simple list without a detail pane (for terminals <60 cols).
func (m inspectModel) viewNarrow(b *strings.Builder) {
	rows := m.contentRows()
	end := min(m.listOffset+rows, len(m.predictions))
	for i := m.listOffset; i < end; i++ {
		p := m.predictions[i]
		timeStr := formatPredictedTime(p.PublishAt)
		if p.Approximate {
			timeStr = "~" + timeStr
		}
		msg := truncate(firstLine(p.Entry.Message), max(m.width-26, 10))

		line := fmt.Sprintf("  %-4d %-19s %s", p.Position, timeStr, msg)
		if i == m.cursor {
			sel := "> " + line[2:]
			if m.confirmDelete {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render(sel))
			} else {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Render(sel))
			}
		} else {
			b.WriteString(inspectDimStyle.Render(line))
		}
		b.WriteString("\n")
	}
	// Pad remaining rows so the alt screen fills.
	for i := end - m.listOffset; i < rows; i++ {
		b.WriteString("\n")
	}
}

// viewSplit renders the horizontal split layout: list | separator | detail.
func (m inspectModel) viewSplit(b *strings.Builder) {
	rows := m.contentRows()
	rightW := m.rightPaneWidth()

	// Top border: ─────┬─────
	b.WriteString(inspectDimStyle.Render(
		strings.Repeat("─", inspectLeftPaneWidth) + "─┬─" + strings.Repeat("─", rightW)))
	b.WriteString("\n")

	// Build left pane (list items padded to leftPaneWidth).
	leftStyle := lipgloss.NewStyle().Width(inspectLeftPaneWidth)
	leftLines := make([]string, rows)
	for i := range rows {
		idx := m.listOffset + i
		if idx < len(m.predictions) {
			leftLines[i] = m.renderListItem(idx, leftStyle)
		} else {
			leftLines[i] = leftStyle.Render("")
		}
	}

	// Build separator column.
	sepColor := lipgloss.Color("240")
	if m.focusDetail {
		sepColor = lipgloss.Color("212")
	}
	sep := lipgloss.NewStyle().Foreground(sepColor).Render(" │ ")

	// Right pane: fixed header + divider + viewport lines.
	p := m.predictions[m.cursor]
	timeStr := formatPredictedTime(p.PublishAt)
	idShort := p.Entry.ID
	if len(idShort) > 8 {
		idShort = idShort[:8]
	}
	header := inspectDimStyle.Render(
		fmt.Sprintf("#%d · %s · %s", p.Position, timeStr, idShort))
	divider := inspectDimStyle.Render(strings.Repeat("─", rightW))

	vpLines := strings.Split(m.detailViewport.View(), "\n")

	// Compose rows: left | sep | right.
	for i := range rows {
		b.WriteString(leftLines[i])
		b.WriteString(sep)
		switch i {
		case 0:
			b.WriteString(header)
		case 1:
			b.WriteString(divider)
		default:
			vpIdx := i - 2
			if vpIdx < len(vpLines) {
				b.WriteString(vpLines[vpIdx])
			}
		}
		b.WriteString("\n")
	}

	// Bottom border: ─────┴─────
	b.WriteString(inspectDimStyle.Render(
		strings.Repeat("─", inspectLeftPaneWidth) + "─┴─" + strings.Repeat("─", rightW)))
	b.WriteString("\n")
}

// renderListItem renders a single list entry for the left pane.
func (m inspectModel) renderListItem(idx int, baseStyle lipgloss.Style) string {
	p := m.predictions[idx]
	timeStr := formatPredictedTime(p.PublishAt)
	if p.Approximate {
		timeStr = "~" + timeStr
	}
	content := fmt.Sprintf("%d  %s", p.Position, timeStr)

	if idx == m.cursor {
		color := lipgloss.Color("212")
		if m.confirmDelete {
			color = lipgloss.Color("214")
		}
		return baseStyle.Foreground(color).Bold(true).Render("> " + content)
	}
	return baseStyle.Foreground(lipgloss.Color("240")).Render("  " + content)
}

func (m inspectModel) helpText() string {
	if m.confirmDelete {
		return "y: confirm   n: cancel"
	}
	if m.width < minSplitWidth {
		return "↑↓: navigate   d: delete   q: quit"
	}
	if m.focusDetail {
		return "↑↓: scroll   tab: list   d: delete   q: quit"
	}
	return "↑↓: navigate   tab: detail   d: delete   q: quit"
}
