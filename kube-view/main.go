package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type model struct {
	k8sClient      *K8sClient
	nodes          []NodeInfo
	width          int
	height         int
	err            error
	loading        bool
	lastUpdate     time.Time
	selectedNode   int
	selectedPod    int
	showingDetails bool
}

// tickMsg is sent on each timer tick for refresh
type tickMsg time.Time

// clusterDataMsg is sent when cluster data is fetched
type clusterDataMsg struct {
	nodes []NodeInfo
	err   error
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	nodeBoxReadyStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#04B575")).
				Padding(0, 1).
				Margin(0, 1, 0, 0)

	nodeBoxNotReadyStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF4040")).
				Padding(0, 1).
				Margin(0, 1, 0, 0)

	nodeBoxSelectedStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Margin(0, 1, 0, 0)

	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Margin(1, 0)

	podBoxRunningStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#04B575")).
				Foreground(lipgloss.Color("#FAFAFA")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center)

	podBoxPendingStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FFA500")).
				Foreground(lipgloss.Color("#000000")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center)

	podBoxFailedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF4040")).
				Foreground(lipgloss.Color("#FAFAFA")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center)

	podBoxSucceededStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#888888")).
				Foreground(lipgloss.Color("#FAFAFA")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center)

	podBoxUnknownStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#666666")).
				Foreground(lipgloss.Color("#FAFAFA")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center)

	podBoxSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FFFF00")).
				Foreground(lipgloss.Color("#000000")).
				Padding(0).
				Width(3).
				Height(1).
				Align(lipgloss.Center).
				Bold(true).
				Blink(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4040")).
			Bold(true)
)

func initialModel() model {
	return model{
		loading:        true,
		lastUpdate:     time.Now(),
		selectedNode:   -1,
		selectedPod:    -1,
		showingDetails: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchClusterData,
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, fetchClusterData
		case "esc":
			m.showingDetails = false
			m.selectedNode = -1
			m.selectedPod = -1
		case "enter", " ":
			if m.selectedNode >= 0 && m.selectedNode < len(m.nodes) {
				m.showingDetails = !m.showingDetails
				// When entering details mode, select first pod
				if m.showingDetails && len(m.nodes[m.selectedNode].Pods) > 0 {
					m.selectedPod = 0
				} else {
					m.selectedPod = -1
				}
			}
		case "left", "h":
			if m.showingDetails && m.selectedPod >= 0 {
				m.selectedPod--
				if m.selectedPod < 0 {
					m.selectedPod = 0
				}
			} else if m.selectedNode > 0 {
				m.selectedNode--
				m.selectedPod = -1
				m.showingDetails = false
			}
		case "right", "l":
			if m.showingDetails && m.selectedNode >= 0 && m.selectedNode < len(m.nodes) {
				if m.selectedPod < len(m.nodes[m.selectedNode].Pods)-1 {
					m.selectedPod++
				}
			} else if m.selectedNode < len(m.nodes)-1 {
				m.selectedNode++
				m.selectedPod = -1
				m.showingDetails = false
			} else if m.selectedNode == -1 && len(m.nodes) > 0 {
				m.selectedNode = 0
			}
		case "up", "k":
			if m.showingDetails {
				// Navigate pods upward
				podsPerRow := 10
				newPod := m.selectedPod - podsPerRow
				if newPod >= 0 {
					m.selectedPod = newPod
				}
			}
		case "down", "j":
			if m.showingDetails && m.selectedNode >= 0 && m.selectedNode < len(m.nodes) {
				// Navigate pods downward
				podsPerRow := 10
				newPod := m.selectedPod + podsPerRow
				if newPod < len(m.nodes[m.selectedNode].Pods) {
					m.selectedPod = newPod
				}
			} else if m.selectedNode == -1 && len(m.nodes) > 0 {
				m.selectedNode = 0
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(
			fetchClusterData,
			tickCmd(),
		)

	case clusterDataMsg:
		m.nodes = msg.nodes
		m.err = msg.err
		m.loading = false
		m.lastUpdate = time.Now()

		// Reset selection if out of bounds
		if m.selectedNode >= len(m.nodes) {
			m.selectedNode = -1
			m.selectedPod = -1
			m.showingDetails = false
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress 'q' to quit, 'r' to retry", m.err))
	}

	if m.loading && len(m.nodes) == 0 {
		return infoStyle.Render("Loading cluster data...\n\nPress 'q' to quit")
	}

	// Build the view
	var view string

	// Title
	title := titleStyle.Render("Kubernetes Cluster View")

	helpText := " ←→/hl: Select Node | Enter: Details | Esc: Back | r: Refresh | q: Quit "
	info := infoStyle.Render(fmt.Sprintf("Last updated: %s | %s",
		m.lastUpdate.Format("15:04:05"), helpText))
	view += title + "\n" + info + "\n\n"

	// Cluster summary
	totalPods := 0
	runningPods := 0
	for _, node := range m.nodes {
		totalPods += len(node.Pods)
		for _, pod := range node.Pods {
			if pod.Status == "Running" {
				runningPods++
			}
		}
	}

	summary := fmt.Sprintf("Nodes: %d | Pods: %d (Running: %d)",
		len(m.nodes), totalPods, runningPods)
	view += infoStyle.Render(summary) + "\n\n"

	// Calculate available height for nodes
	usedHeight := 7 // title + info + summary + spacing
	detailPanelHeight := 0
	if m.showingDetails && m.selectedNode >= 0 && m.selectedNode < len(m.nodes) {
		detailPanelHeight = 11 // detail panel height (actual measured)
	}
	availableHeight := m.height - usedHeight - detailPanelHeight

	// Ensure we have a minimum height
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Display nodes in grid layout
	nodesView := renderNodesGrid(m.nodes, m.width, availableHeight, m.selectedNode, m.selectedPod, m.showingDetails)
	view += nodesView

	// Display detail panel if a node is selected
	if m.showingDetails && m.selectedNode >= 0 && m.selectedNode < len(m.nodes) {
		view += "\n" + renderDetailPanel(m.nodes[m.selectedNode], m.selectedPod)
	}

	// Ensure we don't exceed terminal height by truncating if needed
	lines := strings.Split(view, "\n")
	maxLines := m.height - 1
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		view = strings.Join(lines, "\n")
	}

	return view
}

func renderNodesGrid(nodes []NodeInfo, terminalWidth int, availableHeight int, selectedNode int, selectedPod int, showingDetails bool) string {
	if len(nodes) == 0 {
		return infoStyle.Render("No nodes found")
	}

	// Calculate how many nodes can fit per row
	// Each node box is 34 chars + 2 (padding) + 2 (border) + 2 (margin) = ~40 chars
	const nodeBoxTotalWidth = 40
	nodesPerRow := terminalWidth / nodeBoxTotalWidth
	if nodesPerRow < 1 {
		nodesPerRow = 1
	}

	// Calculate how many rows can fit in available height
	// Each node box: 8 (content) + 2 (border) = 10 lines actual height
	const nodeBoxTotalHeight = 10
	maxRows := availableHeight / nodeBoxTotalHeight
	if maxRows < 1 {
		maxRows = 1
	}

	// Calculate max nodes to display
	maxNodesToDisplay := nodesPerRow * maxRows

	// Limit nodes to display
	nodesToDisplay := nodes
	if len(nodes) > maxNodesToDisplay {
		nodesToDisplay = nodes[:maxNodesToDisplay]
	}

	var rows []string
	var currentRow []string

	for i, node := range nodesToDisplay {
		isSelected := i == selectedNode
		// Pass selectedPod only if this node is selected and showing details
		podToHighlight := -1
		if isSelected && showingDetails {
			podToHighlight = selectedPod
		}
		nodeBox := renderNodeBox(node, isSelected, podToHighlight)
		currentRow = append(currentRow, nodeBox)

		// Start a new row when we hit the limit or it's the last node
		if len(currentRow) == nodesPerRow || i == len(nodesToDisplay)-1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, currentRow...))
			currentRow = []string{}
		}
	}

	result := lipgloss.JoinVertical(lipgloss.Left, rows...)

	// Add info if not all nodes are displayed
	if len(nodes) > maxNodesToDisplay {
		result += "\n" + infoStyle.Render(fmt.Sprintf("... and %d more nodes (resize terminal to see more)", len(nodes)-maxNodesToDisplay))
	}

	return result
}

func renderNodeBox(node NodeInfo, isSelected bool, selectedPod int) string {
	// Select style based on selection and node status
	var boxStyle lipgloss.Style
	if isSelected {
		boxStyle = nodeBoxSelectedStyle
	} else if node.Ready {
		boxStyle = nodeBoxReadyStyle
	} else {
		boxStyle = nodeBoxNotReadyStyle
	}

	// Fixed dimensions for node box - more compact
	const nodeBoxWidth = 34
	const nodeBoxHeight = 8

	// Node header - just the name, more compact
	nodeHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Width(nodeBoxWidth).
		Render(truncateString(node.Name, 20) + fmt.Sprintf("(%d)", len(node.Pods)))

	// Resource usage bars - shorter
	cpuBar := renderResourceBar("C", node.CPUPercent, nodeBoxWidth)
	memBar := renderResourceBar("M", node.MemPercent, nodeBoxWidth)

	// Render pods as colored boxes - more compact
	podsGrid := renderPodsGrid(node.Pods, selectedPod, 8) // 8 pods per row

	// Combine node information - minimal text
	content := nodeHeader + "\n" + cpuBar + "\n" + memBar + "\n" + podsGrid

	// Apply fixed height by padding with empty lines
	lines := strings.Split(content, "\n")
	for len(lines) < nodeBoxHeight {
		lines = append(lines, "")
	}
	// Trim if too many lines
	if len(lines) > nodeBoxHeight {
		lines = lines[:nodeBoxHeight]
	}

	nodeContent := strings.Join(lines, "\n")

	return boxStyle.
		Width(nodeBoxWidth).
		Height(nodeBoxHeight).
		Render(nodeContent)
}

// renderResourceBar renders a horizontal bar chart for resource usage
func renderResourceBar(label string, percent float64, width int) string {
	// Ensure percent is within 0-100
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Calculate bar width - single letter label
	// Account for lipgloss padding (1 left, 1 right) and border (2 total)
	// So actual content width = width - 4
	contentWidth := width - 4
	labelWidth := 1 // "C" or "M"
	availableWidth := contentWidth - labelWidth - 1 // -1 for space after label

	filledCount := int(float64(availableWidth) * percent / 100.0)
	emptyCount := availableWidth - filledCount

	// Color based on usage
	var barColor lipgloss.Color
	if percent < 60 {
		barColor = lipgloss.Color("#04B575") // Green
	} else if percent < 80 {
		barColor = lipgloss.Color("#FFA500") // Orange
	} else {
		barColor = lipgloss.Color("#FF4040") // Red
	}

	// Build bar with ASCII characters (all width-1)
	bar := ""
	for i := 0; i < filledCount; i++ {
		bar += "="
	}
	for i := 0; i < emptyCount; i++ {
		bar += "-"
	}

	barStyle := lipgloss.NewStyle().Foreground(barColor)

	return fmt.Sprintf("%s %s", label, barStyle.Render(bar))
}

func renderPodsGrid(pods []PodInfo, selectedPod int, podsPerRow int) string {
	if len(pods) == 0 {
		return infoStyle.Render("No pods")
	}

	var rows []string
	var currentRow []string

	for i, pod := range pods {
		isSelected := i == selectedPod
		podBox := renderPodBox(pod, isSelected)
		currentRow = append(currentRow, podBox)

		// Start a new row when we hit the limit or it's the last pod
		if len(currentRow) == podsPerRow || i == len(pods)-1 {
			rowStr := lipgloss.JoinHorizontal(lipgloss.Top, currentRow...)
			rows = append(rows, rowStr)
			currentRow = []string{}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderPodBox(pod PodInfo, isSelected bool) string {
	// Select style based on selection or pod status
	var boxStyle lipgloss.Style
	var symbol string

	if isSelected {
		boxStyle = podBoxSelectedStyle
		symbol = "▶" // Arrow to indicate selection
	} else {
		switch pod.Status {
		case "Running":
			boxStyle = podBoxRunningStyle
			symbol = "■"
		case "Pending":
			boxStyle = podBoxPendingStyle
			symbol = "◯"
		case "Failed":
			boxStyle = podBoxFailedStyle
			symbol = "✗"
		case "Succeeded":
			boxStyle = podBoxSucceededStyle
			symbol = "■"
		default:
			boxStyle = podBoxUnknownStyle
			symbol = "■"
		}
	}

	return boxStyle.Render(symbol)
}

func renderDetailPanel(node NodeInfo, selectedPod int) string {
	var content string

	// Node details
	content += lipgloss.NewStyle().Bold(true).Render("Node: "+node.Name) + "\n"
	content += fmt.Sprintf("Status: %s\n", node.Status)
	content += fmt.Sprintf("CPU: %s\n", node.CPUUsage)
	content += fmt.Sprintf("Memory: %s\n", node.MemUsage)
	content += fmt.Sprintf("Total Pods: %d\n", len(node.Pods))

	// Pod details if selected
	if selectedPod >= 0 && selectedPod < len(node.Pods) {
		pod := node.Pods[selectedPod]
		content += "\n" + lipgloss.NewStyle().Bold(true).Render("Selected Pod:") + "\n"
		content += fmt.Sprintf("Name: %s\n", pod.Name)
		content += fmt.Sprintf("Namespace: %s\n", pod.Namespace)
		content += fmt.Sprintf("Status: %s\n", pod.Status)
		content += fmt.Sprintf("Ready: %v\n", pod.Ready)
	} else {
		content += "\n" + infoStyle.Render("Use ←→↑↓ or hjkl to select a pod")
	}

	return detailPanelStyle.Render(content)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// wrapText wraps text to a specified width
func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// calculatePodsPerRow calculates how many pods can fit per row based on available width
func calculatePodsPerRow(availableWidth int) int {
	// Each pod box is approximately 12 characters wide (with margin)
	podWidth := 12
	podsPerRow := availableWidth / podWidth
	if podsPerRow < 1 {
		podsPerRow = 1
	}
	return podsPerRow
}

func fetchClusterData() tea.Msg {
	// This will be called from Init, so we need to create the client here
	k8sClient, err := NewK8sClient()
	if err != nil {
		return clusterDataMsg{err: err}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes, err := k8sClient.GetClusterData(ctx)
	return clusterDataMsg{nodes: nodes, err: err}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
