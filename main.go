package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	apiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent"
)

var apiKey = os.Getenv("GEMINI_API_KEY")

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func handleSession(s ssh.Session) {
	fmt.Fprintln(s, "Connected to Gemini Flash via SSH.")
	fmt.Fprint(s, "> ")

	buf := make([]byte, 4096)
	n, err := s.Read(buf)
	if err != nil {
		fmt.Fprintln(s, "Failed to read input")
		s.Exit(1)
		return
	}
	prompt := strings.TrimSpace(string(buf[:n]))

	resp, err := queryGemini(prompt)
	if err != nil {
		fmt.Fprintf(s, "Error: %v\n", err)
		s.Exit(1)
		return
	}

	fmt.Fprintln(s, "\n--- Gemini Response ---")
	fmt.Fprintln(s, resp)
	s.Exit(0)
}

func queryGemini(prompt string) (string, error) {
	request := GeminiRequest{
		Contents: []Content{
			{
				Role: "user",
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		respBody, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("HTTP %d: %s", res.StatusCode, string(respBody))
	}

	var gemResp GeminiResponse
	err = json.NewDecoder(res.Body).Decode(&gemResp)
	if err != nil {
		return "", err
	}
	if len(gemResp.Candidates) == 0 {
		return "(no response)", nil
	}
	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

func main() {
	if apiKey == "" {
		log.Fatal("Set GEMINI_API_KEY in env")
	}

	privateBytes, err := os.ReadFile("id_rsa") // generate if missing
	if err != nil {
		log.Fatal("Missing SSH key. Run: ssh-keygen -t rsa -f id_rsa")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse SSH key:", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(private)

	listener, err := net.Listen("tcp", ":2222")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Println("SSH Gemini gateway running on port 2222")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go func() {
			sshConn, chans, _, err := ssh.NewServerConn(conn, config)
			if err != nil {
				log.Printf("SSH handshake failed: %v", err)
				return
			}
			defer sshConn.Close()

			for newChannel := range chans {
				if newChannel.ChannelType() != "session" {
					newChannel.Reject(ssh.UnknownChannelType, "Only session supported")
					continue
				}
				channel, requests, _ := newChannel.Accept()
				go ssh.DiscardRequests(requests)
				handleSession(channel)
			}
		}()
	}
}
