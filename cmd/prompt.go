// Copyright (c) 2026 Beijing Volcano Engine Technology Co., Ltd.
// SPDX-License-Identifier: MIT
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type promptItem struct {
	summary string
	details string
	value   string
}

func (i promptItem) Title() string       { return i.summary }
func (i promptItem) Description() string { return i.details }
func (i promptItem) FilterValue() string { return i.summary + " " + i.details }

type promptDelegate struct{}

func (promptDelegate) Height() int                         { return 1 }
func (promptDelegate) Spacing() int                        { return 0 }
func (promptDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }
func (promptDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(promptItem)
	if !ok {
		return
	}
	text := fmt.Sprintf("%d. %s", index+1, i.summary)
	if i.details != "" {
		text += " [" + i.details + "]"
	}
	style := lipgloss.NewStyle().PaddingLeft(4)
	if index == m.Index() {
		style = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
		text = "> " + text
	}
	_, _ = fmt.Fprint(w, style.Render(text))
}

type promptModel struct {
	cancel context.CancelFunc
	list   list.Model
	choice string
}

func (m promptModel) Init() tea.Cmd { return nil }

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancel()
			return m, tea.Quit
		case tea.KeyEnter:
			if item, ok := m.list.SelectedItem().(promptItem); ok {
				m.choice = item.summary
				if item.value != "" {
					m.choice = item.value
				}
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func promptText(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("search input cannot be empty")
	}
	return input, nil
}

func (m promptModel) View() string {
	if m.choice != "" {
		return ""
	}
	return "\n" + m.list.View()
}

func promptList(ctx context.Context, title string, items []promptItem) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no choices available")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	listItems := make([]list.Item, len(items))
	for i := range items {
		listItems[i] = items[i]
	}
	l := list.New(listItems, promptDelegate{}, 0, minInt(len(items)*4, 14))
	l.Title = title
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2)
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	state, err := tea.NewProgram(promptModel{cancel: cancel, list: l}, tea.WithOutput(os.Stderr)).Run()
	if err != nil {
		return "", fmt.Errorf("failed to prompt choice: %w", err)
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	choice := state.(promptModel).choice
	if choice == "" {
		return "", fmt.Errorf("user aborted")
	}
	return choice, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
