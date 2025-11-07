package tui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type gentooUseFlagsSetMsg struct {
	err error
}

func (m Model) viewGentooUseFlags() string {
	var b strings.Builder

	b.WriteString(m.renderBanner())
	b.WriteString("\n")

	title := m.styles.Title.Render("Gentoo Global USE Flags")
	b.WriteString(title)
	b.WriteString("\n\n")

	info := m.styles.Normal.Render("The following global USE flags will be enabled in /etc/portage/make.conf:")
	b.WriteString(info)
	b.WriteString("\n\n")

	useFlags := []string{
		"wayland",
		"vulkan",
		"opengl",
		"accessibility",
		"policykit",
		"X",
		"udev",
		"alsa",
		"gdbm",
		"qml",
		"gtk",
		"qt6",
		"jpeg",
		"webp",
		"png",
		"gif",
		"tiff",
		"brotli",
		"dbus",
		"svg",
	}

	for _, flag := range useFlags {
		flagLine := m.styles.Success.Render(fmt.Sprintf("  • %s", flag))
		b.WriteString(flagLine)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	note := m.styles.Subtle.Render("These flags ensure proper Qt6, Wayland, and compositor support.")
	b.WriteString(note)
	b.WriteString("\n\n")

	help := m.styles.Subtle.Render("Press Enter to continue, Esc to go back")
	b.WriteString(help)

	return b.String()
}

func (m Model) updateGentooUseFlagsState(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			if checkFingerprintEnabled() {
				m.state = StateAuthMethodChoice
				m.selectedConfig = 0
			} else {
				m.state = StatePasswordPrompt
				m.passwordInput.Focus()
			}
			return m, nil
		case "esc":
			m.state = StateDependencyReview
			return m, nil
		}
	}
	return m, m.listenForLogs()
}

func (m Model) setGentooGlobalUseFlags() tea.Cmd {
	return func() tea.Msg {
		useFlags := "wayland vulkan opengl accessibility policykit X"

		// Check if USE line already exists
		checkCmd := exec.CommandContext(context.Background(), "grep", "-q", "^USE=", "/etc/portage/make.conf")
		hasUse := checkCmd.Run() == nil

		var cmd *exec.Cmd
		if hasUse {
			// Append to existing USE flags
			cmdStr := fmt.Sprintf("echo '%s' | sudo -S sed -i 's/^USE=\"\\(.*\\)\"/USE=\"\\1 %s\"/' /etc/portage/make.conf", m.sudoPassword, useFlags)
			cmd = exec.CommandContext(context.Background(), "bash", "-c", cmdStr)
		} else {
			// Add new USE line
			cmdStr := fmt.Sprintf("echo '%s' | sudo -S bash -c \"echo 'USE=\\\"%s\\\"' >> /etc/portage/make.conf\"", m.sudoPassword, useFlags)
			cmd = exec.CommandContext(context.Background(), "bash", "-c", cmdStr)
		}

		err := cmd.Run()
		return gentooUseFlagsSetMsg{err: err}
	}
}
