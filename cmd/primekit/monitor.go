package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"github.com/ishaandevburman/primekit/pkg/algo"

	"github.com/ishaandevburman/primekit/pkg/store"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	stateIdle = iota
	stateRunning
)

type progressMsg struct {
	p algo.Progress
}

type doneMsg struct {
	label   string
	elapsed time.Duration
	err     error
}

type statsMsg struct {
	storeCount uint64
	storeMax   uint64
	segments   int
	benchmarks int
}

type model struct {
	cfg  config
	prog *tea.Program

	state       int
	spinner     spinner.Model
	pbar        progress.Model
	viewport    viewport.Model

	stats        statsMsg
	activeLabel  string
	segmentsDone int
	totalSegments int
	primesFound  uint64
	end          uint64

	output []string
	width  int
	height int
}

func cmdMonitor(ctx context.Context, cfg config) {
	m := model{
		cfg:      cfg,
		spinner:  newSpinner(),
		pbar:     progress.New(progress.WithDefaultGradient()),
		viewport: newViewport(),
		output:   make([]string, 0, 100),
	}
	p := tea.NewProgram(&m, tea.WithAltScreen())
	m.prog = p
	if _, err := p.Run(); err != nil {
		fail("monitor: %v", err)
	}
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	s.Spinner = spinner.Dot
	return s
}

func newViewport() viewport.Model {
	vp := viewport.New(60, 8)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	return vp
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.pollStats)
}

func (m model) pollStats() tea.Msg {
	var s statsMsg
	st, err := store.NewBinaryStore(m.cfg.storePath)
	if err == nil {
		s.storeCount = st.Count()
		s.storeMax = st.MaxPrime()
		st.Close()
	}
	sq, err := store.NewSQLiteStore(m.cfg.dbPath)
	if err == nil {
		ctx := context.Background()
		segs, _ := sq.ListSegments(ctx)
		s.segments = len(segs)
		bencs, _ := sq.ListBenchmarks(ctx)
		s.benchmarks = len(bencs)
		sq.Close()
	}
	return s
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 14
		if m.viewport.Height < 3 {
			m.viewport.Height = 3
		}
		m.pbar.Width = msg.Width - 20
		if m.pbar.Width < 10 {
			m.pbar.Width = 10
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			if m.state == stateIdle {
				m.state = stateRunning
				m.activeLabel = "sieving up to 10⁷"
				m.segmentsDone = 0
				m.totalSegments = 0
				m.primesFound = 0
				m.end = 0
				go m.runSieve(10_000_000)
			}
		case "2":
			if m.state == stateIdle {
				m.state = stateRunning
				m.activeLabel = "sieving up to 10⁸"
				m.segmentsDone = 0
				m.totalSegments = 0
				m.primesFound = 0
				m.end = 0
				go m.runSieve(100_000_000)
			}
		case "3":
			if m.state == stateIdle {
				m.state = stateRunning
				m.activeLabel = "running quick bench"
				m.segmentsDone = 0
				m.totalSegments = 0
				m.primesFound = 0
				m.end = 0
				go m.runBench()
			}
		case "r":
			if m.state == stateIdle {
				m.output = m.output[:0]
				m.viewport.SetContent("")
				return m, nil
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progressMsg:
		p := msg.p
		m.segmentsDone = p.SegmentsDone
		m.totalSegments = p.TotalSegments
		m.primesFound = p.PrimesFound
		m.end = p.End
		pct := float64(p.SegmentsDone) / float64(p.TotalSegments)
		cmd := m.pbar.SetPercent(pct)
		return m, cmd

	case doneMsg:
		m.state = stateIdle
		line := fmt.Sprintf("✓ %s in %v", msg.label, msg.elapsed.Round(time.Millisecond))
		if msg.err != nil {
			line = fmt.Sprintf("✗ %s: %v", msg.label, msg.err)
		}
		m.output = append(m.output, line)
		m.viewport.SetContent(strings.Join(m.output, "\n"))
		m.viewport.GotoBottom()
		m.pbar.SetPercent(0)
		m.segmentsDone = 0
		m.totalSegments = 0
		m.primesFound = 0
		m.end = 0
		return m, nil

	case statsMsg:
		m.stats = msg
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return m.pollStats()
		})
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00")).
		Render(" primekit monitor ")

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Render(title))
	b.WriteString("\n\n")

	st := m.stats
	countStr := "—"
	if st.storeCount > 0 {
		countStr = fmt.Sprintf("%d", st.storeCount)
	}
	maxStr := "—"
	if st.storeMax > 0 {
		maxStr = fmt.Sprintf("%d", st.storeMax)
	}
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	b.WriteString(statsStyle.Render(fmt.Sprintf(
		" Store: %s primes  max: %s  |  DB: %d segments, %d benchmarks",
		countStr, maxStr, st.segments, st.benchmarks)))
	b.WriteString("\n\n")

	controlsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	b.WriteString(controlsStyle.Render(
		" [1] Sieve 10⁷  [2] Sieve 10⁸  [3] Bench  [r] Clear  [q] Quit"))
	b.WriteString("\n\n")

	if m.state == stateRunning {
		activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		b.WriteString(activeStyle.Render(fmt.Sprintf(" %s %s",
			m.spinner.View(), m.activeLabel)))
		b.WriteString("\n")

		pct := 0.0
		if m.totalSegments > 0 {
			pct = float64(m.segmentsDone) / float64(m.totalSegments)
		}
		b.WriteString(" " + m.pbar.ViewAs(pct) + "\n")

		detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		b.WriteString(detailStyle.Render(fmt.Sprintf(
			" segments: %d/%d  primes: %d  limit: %d",
			m.segmentsDone, m.totalSegments, m.primesFound, m.end)))
		b.WriteString("\n\n")
	}

	b.WriteString(lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Render(m.viewport.View()))
	b.WriteString("\n")

	if m.state == stateIdle {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
		b.WriteString(helpStyle.Render(" idle — press 1/2/3 to start"))
	}

	return b.String()
}

func (m model) runSieve(limit uint64) {
	prog := m.prog
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	gen := algo.NewSegmentedSieve(1 << 20)
	gen.OnProgress = func(p algo.Progress) {
		prog.Send(progressMsg{p: p})
	}

	out := make(chan uint64, 65536)
	go func() {
		for range out {
		}
	}()

	err := gen.PrimesInRange(ctx, 2, limit, out)
	prog.Send(doneMsg{
		label:   fmt.Sprintf("sieve(%d)", limit),
		elapsed: time.Since(start),
		err:     err,
	})
}

func (m model) runBench() {
	prog := m.prog
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()

	suite := algo.NewBenchmarkSuite()
	suite.RunSieve(ctx, []uint64{100000})
	prog.Send(doneMsg{
		label:   "benchmarks",
		elapsed: time.Since(start),
	})
}
