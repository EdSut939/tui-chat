package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gorilla/websocket"
)

const wsURL = "ws://localhost:8080/ws"

func dialServer() tea.Cmd {
	return func() tea.Msg {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			return wsErrMsg(err)
		}
		return connectMsg{conn: c}
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oof: %v\n", err)
	}
}

type model struct {
	wsConn        *websocket.Conn
	viewport      viewport.Model
	messages      []string
	textarea      textarea.Model
	senderStyle   lipgloss.Style
	receiverStyle lipgloss.Style
	err           error
}

type connectMsg struct {
	conn *websocket.Conn
}

type wsMsg []byte

type wsErrMsg error

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.SetVirtualCursor(false)
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 280

	ta.SetWidth(30)
	ta.SetHeight(3)

	// Remove cursor line styling
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	ta.ShowLineNumbers = false

	vp := viewport.New(viewport.WithWidth(30), viewport.WithHeight(5))
	vp.SetContent(`Welcome to the chat room!
Type a message and press Enter to send.`)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	ta.KeyMap.InsertNewline.SetEnabled(false)

	return model{
		textarea:      ta,
		messages:      []string{},
		viewport:      vp,
		senderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		receiverStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		err:           nil,
	}
}

func (m *model) refreshViewport() {
	content := lipgloss.NewStyle().
		Width(m.viewport.Width()).
		Render(strings.Join(m.messages, "\n"))
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m model) Init() tea.Cmd {
	return dialServer()
}

func readNextWsMessage(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return wsErrMsg(err)
		}
		return wsMsg(message)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectMsg:
		m.wsConn = msg.conn
		return m, readNextWsMessage(m.wsConn)

	case wsMsg:
		m.messages = append(m.messages, m.receiverStyle.Render("Server: ")+string(msg))
		m.refreshViewport()
		return m, readNextWsMessage(m.wsConn)

	case wsErrMsg:
		m.err = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
		m.textarea.SetWidth(msg.Width)
		m.viewport.SetHeight(msg.Height - m.textarea.Height())

		if len(m.messages) > 0 {
			// Wrap content before setting it.
			m.viewport.SetContent(lipgloss.NewStyle().Width(m.viewport.Width()).Render(strings.Join(m.messages, "\n")))
		}
		m.viewport.GotoBottom()

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case "enter":
			text := m.textarea.Value()
			if text == "" {
				return m, nil
			}

			if m.wsConn != nil {
				err := m.wsConn.WriteMessage(websocket.TextMessage, []byte(text))
				if err != nil {
					m.err = err
					return m, tea.Quit
				}
			}

			m.messages = append(m.messages, m.senderStyle.Render("You: ")+text)
			m.textarea.Reset()
			m.refreshViewport()
			return m, nil
		default:
			// Send all other keypresses to the textarea.
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	viewportView := m.viewport.View()
	v := tea.NewView(viewportView + "\n" + m.textarea.View())
	c := m.textarea.Cursor()
	if c != nil {
		c.Y += lipgloss.Height(viewportView)
	}
	v.Cursor = c
	v.AltScreen = true
	return v
}
