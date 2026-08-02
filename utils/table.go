package utils

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

var (
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.ANSIColor(15)).
		Padding(0, 1)

	cellStyle = lipgloss.NewStyle().
		Foreground(lipgloss.ANSIColor(7)).
		Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.ANSIColor(8))
)

// PrintTable prints a formatted table with headers and rows
// In AI mode, renders as a markdown table for easy parsing
// Note: table.HeaderRow == -1, data rows start at 0
func PrintTable(headers []string, rows [][]string) {
	if GlobalForAIFlag {
		printMarkdownTable(headers, rows)
		return
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(borderStyle).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	PrintGeneric(t.Render())
}

// printMarkdownTable renders headers and rows as a markdown table
func printMarkdownTable(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	fmt.Println("| " + strings.Join(escapeCells(headers), " | ") + " |")
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Println("| " + strings.Join(seps, " | ") + " |")
	for _, row := range rows {
		fmt.Println("| " + strings.Join(escapeCells(row), " | ") + " |")
	}
}

// escapeCells escapes pipe characters in cell values for valid markdown tables
func escapeCells(cells []string) []string {
	escaped := make([]string, len(cells))
	for i, cell := range cells {
		escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
	}
	return escaped
}
