package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sai-mudike/git-commit-ai/models"
)

func main() {
	cmd := exec.Command("git", "diff", "--staged")
	diff, err := cmd.Output()

	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get staged changes:", err)
		os.Exit(1)
	}
	output := strings.TrimSpace(string(diff))

	if output == "" {
		fmt.Println("No staged changes found.")
		return
	}
	start := time.Now()

	commitMSG, err := generateCommitMessage(output)

	fmt.Printf("Time took for generation:%v seconds\n", time.Since(start).Abs().Seconds())

	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to generate commit message:", err)
		os.Exit(1)
	}

	if !validateCommitMessage(commitMSG) {
		fmt.Fprintln(os.Stderr, "AI generated an invalid commit message:")
		fmt.Fprintln(os.Stderr, commitMSG)
		os.Exit(1)

	}
	fmt.Println("Suggested commit")
	// fmt.Println(output)
	fmt.Println(commitMSG)
}

func generateCommitMessage(diff string) (string, error) {
	prompt := `You are a Git commit message generator.

Analyze the Git diff below and generate exactly ONE Conventional Commit message.

Format:
<type>: <short description>

Allowed types:
feat, fix, refactor, docs, test, chore, style, perf

Rules:
- Return only the commit message.
- Do not explain your answer.
- Do not use markdown.
- Do not use quotes.
- Keep the description short and specific.
- Use lowercase for the type.

Example:
feat: add JWT authentication middleware

Git diff:
` + diff
	requestData := models.OllamaRequest{
		Model:  "qwen2.5:1.5b",
		Prompt: prompt,
		Stream: false,
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		return "", err
	}
	res, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf(
			"Ollama returned status %d: %s",
			res.StatusCode,
			string(body),
		)
	}

	var ollamaResponse models.OllamaResponse
	err = json.NewDecoder(res.Body).Decode(&ollamaResponse)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(ollamaResponse.Response), nil
}

func validateCommitMessage(s string) bool {

	commit := strings.ToLower(strings.TrimSpace(s))

	allowedTypes := []string{
		"feat",
		"fix",
		"refactor",
		"docs",
		"test",
		"chore",
		"style",
		"perf",
	}

	for _, commitType := range allowedTypes {

		prefix := commitType + ":"
		if strings.HasPrefix(commit, prefix) {
			return true
		}
		// scopePrefix := commitType + "("
		// if strings.HasPrefix(commitType, scopePrefix) &&
		// 	strings.Contains(commitType, "):") {
		// 	return true
		// }
	}
	return false
}
