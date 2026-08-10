package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// stdinScanner is shared across calls so sequential PromptInput/PromptPassword
// calls each read the next line instead of draining all of stdin on the first call
var stdinScanner *bufio.Scanner

func getStdinScanner() *bufio.Scanner {
	if stdinScanner == nil {
		if fi, err := os.Stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice == 0 {
			stdinScanner = bufio.NewScanner(os.Stdin)
		}
	}
	return stdinScanner
}

func ReadPipedInput() string {
	scanner := getStdinScanner()
	if scanner == nil {
		return ""
	}
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func ReadPipedLine() string {
	scanner := getStdinScanner()
	if scanner == nil {
		return ""
	}
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

type inputModel struct {
	textInput textinput.Model
	done      bool
	value     string
	initCmd   tea.Cmd
}

func (m inputModel) Init() tea.Cmd {
	return m.initCmd
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			m.value = m.textInput.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textInput.View())
}

func PromptInput(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = prompt + " "
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return strings.TrimSpace(result.value), nil
}

// Security note: caller must ensure the returned value is never passed to Print functions
func PromptPassword(prompt string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedLine(), nil
	}

	ti := textinput.New()
	ti.Placeholder = "••••••••"
	ti.Prompt = prompt + " "
	ti.EchoMode = textinput.EchoPassword
	focusCmd := ti.Focus()

	m := inputModel{textInput: ti, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	return result.value, nil
}

type textAreaModel struct {
	textarea textarea.Model
	done     bool
	value    string
	initCmd  tea.Cmd
}

func (m textAreaModel) Init() tea.Cmd {
	return m.initCmd
}

func (m textAreaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+d":
			m.value = m.textarea.Value()
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textAreaModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	return tea.NewView(m.textarea.View() + "\n(Ctrl+D to submit, Esc to cancel)")
}

func PromptTextArea(prompt string, placeholder string) (string, error) {
	if GlobalForAIFlag {
		return ReadPipedInput(), nil
	}

	PrintInfo(prompt)

	ta := textarea.New()
	ta.Placeholder = placeholder
	focusCmd := ta.Focus()

	m := textAreaModel{textarea: ta, initCmd: focusCmd}
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(textAreaModel)
	return strings.TrimSpace(result.value), nil
}

type selectModel struct {
	label    string
	options  []string
	cursor   int
	selected int
	done     bool
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.selected = -1
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	var b strings.Builder
	if m.label != "" {
		b.WriteString(infoStyle.Render("→ "+m.label) + "\n")
	}
	for i, opt := range m.options {
		cursor := "  "
		item := opt
		if m.cursor == i {
			cursor = "❯ "
			item = infoStyle.Render(opt)
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, item))
	}
	b.WriteString(warnStyle.Render("(Use arrows/j/k to move, Enter to select, Esc to cancel)"))
	return tea.NewView(b.String())
}

func PromptSelect(label string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, nil
	}
	if GlobalForAIFlag {
		line := strings.TrimSpace(ReadPipedLine())
		if line == "" {
			return -1, nil
		}
		idx, err := strconv.Atoi(line)
		if err != nil || idx < 1 || idx > len(options) {
			return -1, nil
		}
		return idx - 1, nil
	}

	m := selectModel{label: label, options: options, cursor: 0, selected: -1}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}
	res := finalModel.(selectModel)
	return res.selected, nil
}

type multiSelectModel struct {
	label    string
	options  []string
	cursor   int
	selected map[int]bool
	done     bool
	canceled bool
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "space":
			if m.selected[m.cursor] {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = true
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.canceled = true
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m multiSelectModel) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	var b strings.Builder
	if m.label != "" {
		b.WriteString(infoStyle.Render("→ "+m.label) + "\n")
	}
	for i, opt := range m.options {
		cursor := "  "
		checked := "[ ] "
		if m.selected[i] {
			checked = "[✓] "
		}
		item := opt
		if m.cursor == i {
			cursor = "❯ "
			item = infoStyle.Render(opt)
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, checked, item))
	}
	b.WriteString(warnStyle.Render("(Space to toggle, Enter to confirm, Esc to cancel)"))
	return tea.NewView(b.String())
}

func PromptMultiSelect(label string, options []string) (map[int]bool, error) {
	if len(options) == 0 {
		return nil, nil
	}
	if GlobalForAIFlag {
		line := strings.TrimSpace(ReadPipedLine())
		if line == "" || line == "none" {
			return nil, nil
		}
		selected := make(map[int]bool)
		parts := strings.Split(line, ",")
		for _, part := range parts {
			idx, err := strconv.Atoi(strings.TrimSpace(part))
			if err == nil && idx >= 1 && idx <= len(options) {
				selected[idx-1] = true
			}
		}
		return selected, nil
	}

	m := multiSelectModel{
		label:    label,
		options:  options,
		cursor:   0,
		selected: make(map[int]bool),
	}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	res := finalModel.(multiSelectModel)
	if res.canceled {
		return nil, nil
	}
	return res.selected, nil
}
