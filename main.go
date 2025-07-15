package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	cssh "github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bbtea "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

const apiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent"

var apiKey = os.Getenv("GEMINI_API_KEY")

type model struct {
	prompt  string
	output  string
	loading bool
	err     error
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if strings.TrimSpace(m.prompt) == "" {
				return m, nil
			}
			m.loading = true
			return m, sendToGemini(m.prompt)
		case tea.KeyBackspace:
			if !m.loading && len(m.prompt) > 0 {
				m.prompt = m.prompt[:len(m.prompt)-1]
			}
		default:
			if !m.loading {
				m.prompt += msg.String()
			}
		}
	case geminiResponse:
		m.loading = false
		m.output = msg.Text
		m.prompt = "" // Clear prompt after response
	case geminiError:
		m.loading = false
		m.err = msg.Err
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress Ctrl+C to quit.", m.err)
	}

	view := "# Gemini 1.5 Flash over SSH\n\n"

	if m.loading {
		view += fmt.Sprintf("Prompt: %s\n\n   Thinking...\n\n", m.prompt)
	} else {
		if m.output != "" {
			view += fmt.Sprintf("Response:\n%s\n\n", m.output)
		}
		view += fmt.Sprintf("> %s\n", m.prompt)
		view += "\nType your prompt and press Enter. Ctrl+C to exit.\n"
	}

	return view
}

// Msg types
type geminiResponse struct{ Text string }
type geminiError struct{ Err error }

func sendToGemini(prompt string) tea.Cmd {
	return func() tea.Msg {
		text, err := queryGemini(prompt)
		if err != nil {
			return geminiError{Err: err}
		}
		return geminiResponse{Text: text}
	}
}

// Gemini API request
func queryGemini(prompt string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY env var")
	}

	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "(no response)", nil
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

func main() {
	if apiKey == "" {
		log.Fatal("Please set GEMINI_API_KEY")
	}

	if os.Getenv("TEST") != "" {
		println("yo moma")
		// test gemini connection
		res, err := queryGemini("hello who are you?")
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println("Result:\n" + res)
		return
	}

	// Handler that returns just the model - options are handled by the middleware
	// type Handler func(sess ssh.Session) (tea.Model, []tea.ProgramOption)

	var handler bbtea.Handler = func(session cssh.Session) (tea.Model, []tea.ProgramOption) {
		return initialModel(), nil
	}

	s, err := wish.NewServer(
		wish.WithAddress(":2222"),
		wish.WithHostKeyPath("id_ed25519"), // run ssh-keygen -t ed25519 -f id_ed25519 if missing
		wish.WithMiddleware(
			logging.Middleware(),
			bbtea.Middleware(handler),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("SSH Gemini is running on port 2222")
	if err := s.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
