package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	// Set up logging to `debug.log` file
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatal("Fatal error:", err)
		}
		defer f.Close()
	}

	// Set up and run bubbletea program
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal("Sorry, an error occurred:", err)
	}
}

type model struct {
	welcomeMsg string
}

func initialModel() model {
	return model{
		welcomeMsg: "Hello world!",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		{
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	s := m.welcomeMsg
	s += "\n\n\nPress `q` or `ctrl+c` to quit.\n"
	return s
}
