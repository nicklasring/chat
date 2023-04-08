package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	markdown "github.com/MichaelMure/go-term-markdown"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model    string     `json:"model"`
	Messages []*Message `json:"messages"`
}

type Response struct {
	Choices []*Choice `json:"choices"`
	Created int64     `json:"created"`
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Object  string    `json:"object"`
	Usage   struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type Choice struct {
	FinishReason string   `json:"finish_reason"`
	Index        int      `json:"index"`
	Message      *Message `json:"message"`
}

func spinner(stop chan bool) {
	symbols := []string{"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"}
	i := 0
	for {
		select {
		case <-stop:
			return
		default:
			fmt.Printf("\r%s", symbols[i])
			i = (i + 1) % len(symbols)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func main() {
	token := os.Getenv("OPENAI_API_TOKEN")
	if token == "" {
		tokenFile, err := os.ReadFile(os.Getenv("HOME") + "/.openai/token")
		if err != nil {
			fmt.Println("Please set either the OPENAI_API_TOKEN environment variable or create the ~/.openai/token file")
			return
		}

		token = strings.TrimSpace(string(tokenFile))
	}

	model := "gpt-3.5-turbo-0301"
	url := "https://api.openai.com/v1/chat/completions"
	request := &Request{
		Model: model,
		Messages: []*Message{
			{Role: "system", Content: "You are a helpful assistant."},
		},
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanLines)

	fmt.Print("> ")

	for {
		var inputLines []string

		for scanner.Scan() {
			line := scanner.Text()
			if line == "END" {
				break
			}
			inputLines = append(inputLines, line)
		}

		stopSpinner := make(chan bool)
		go spinner(stopSpinner)

		if len(inputLines) == 0 {
			fmt.Print("> ")
			continue
		}

		question := strings.Join(inputLines, "\n")

		request.Messages = append(request.Messages, &Message{Role: "user", Content: question})

		body, err := json.Marshal(request)
		if err != nil {
			fmt.Println("Error marshaling request:", err)
			continue
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("Error creating request:", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			continue
		}

		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response:", err)
			continue
		}

		response := &Response{}
		err = json.Unmarshal(respBody, response)
		if err != nil {
			fmt.Println("Error unmarshaling response:", err)
			continue
		}

		if len(response.Choices) > 0 {
			choice := response.Choices[0]
			if choice.Message != nil {
				result := markdown.Render(choice.Message.Content, 80, 0)
				fmt.Printf("\r# %s", string(result))

				request.Messages = append(request.Messages, choice.Message)
			}
		}
		stopSpinner <- true
		fmt.Print("\n> ")
	}

}
