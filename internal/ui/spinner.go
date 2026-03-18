package ui

import (
    "fmt"

    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type spinnerModel struct {
    spinner  spinner.Model
    message  string
    done     bool
    result   interface{}
    err      error
}

type doneMsg struct {
    result interface{}
    err    error
}

func (m spinnerModel) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case doneMsg:
        m.done = true
        m.result = msg.result
        m.err = msg.err
        return m, tea.Quit
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m spinnerModel) View() string {
    if m.done {
        return ""
    }
    return fmt.Sprintf("%s %s\n",
        lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Render(m.spinner.View()),
        m.message)
}

// RunWithSpinner führt fn aus und zeigt einen Spinner bis fn fertig ist.
// Gibt das Ergebnis von fn zurück.
func RunWithSpinner(message string, fn func() (interface{}, error)) (interface{}, error) {
    s := spinner.New()
    s.Spinner = spinner.Dot
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

    m := spinnerModel{spinner: s, message: message}
    p := tea.NewProgram(m)

    go func() {
        result, err := fn()
        p.Send(doneMsg{result: result, err: err})
    }()

    final, err := p.Run()
    if err != nil {
        return nil, err
    }

    sm := final.(spinnerModel)
    return sm.result, sm.err
}
