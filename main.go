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

	suggestion, err := generateCommitMessage(output)

	fmt.Printf("Time took for generation:%.2f seconds\n", time.Since(start).Seconds())

	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to generate commit message:", err)
		os.Exit(1)
	}
	if !isValidCommitType(suggestion.Type) {
		fmt.Fprintln(os.Stderr, "AI returned an invalid commit type:", suggestion.Type)
		os.Exit(1)
	}

	if strings.TrimSpace(suggestion.Description) == "" {
		fmt.Fprintln(os.Stderr, "AI returned an invalid commit description:", suggestion.Type)
		os.Exit(1)
	}
	commitMSG := formatCommitMSG(suggestion)
	fmt.Println("Suggested commit")
	fmt.Println(output)
	fmt.Println(commitMSG)
}

func generateCommitMessage(diff string) (models.CommitSuggestion, error) {
	prompt := `Analyze the following staged Git diff.

Return ONLY valid JSON using exactly this structure:


example:
{
  "type": "feat",
  "scope": "auth",
  "description": "add JWT authentication"
}

Rules:
- type must be one of: feat, fix, refactor, docs, test, chore, style, perf
- scope should be a short word describing the affected area
- description should be short and specific
- do not include markdown
- do not include explanations
- return only the JSON object

Git diff:
` + diff
	requestData := models.OllamaRequest{
		Model:  "qwen2.5:1.5b",
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}
	jsonData, err := json.Marshal(requestData)

	if err != nil {
		return models.CommitSuggestion{}, err
	}
	res, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return models.CommitSuggestion{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return models.CommitSuggestion{}, fmt.Errorf(
			"Ollama returned status %d: %s",
			res.StatusCode,
			string(body),
		)
	}

	var ollamaResponse models.OllamaResponse
	err = json.NewDecoder(res.Body).Decode(&ollamaResponse)
	if err != nil {
		return models.CommitSuggestion{}, err
	}
	fmt.Println(ollamaResponse.Response)
	var suggestion models.CommitSuggestion

	err = json.Unmarshal([]byte(ollamaResponse.Response), &suggestion)
	if err != nil {
		return models.CommitSuggestion{}, err
	}

	return suggestion, nil
}

func isValidCommitType(s string) bool {

	commit := strings.ToLower(strings.TrimSpace(s))

	allowedTypes := map[string]bool{
		"feat":     true,
		"fix":      true,
		"refactor": true,
		"docs":     true,
		"test":     true,
		"chore":    true,
		"style":    true,
		"perf":     true,
	}

	if allowedTypes[commit] == true {
		return true
	}
	return false
}

func formatCommitMSG(response models.CommitSuggestion) string {
	commitType := strings.ToLower(strings.TrimSpace(response.Type))
	scope := strings.ToLower(strings.TrimSpace(response.Scope))
	description := strings.ToLower(strings.TrimSpace(response.Description))
	if scope == "" {
		return fmt.Sprintf("%s: %s", commitType, description)
	}
	return fmt.Sprintf("%s (%s): %s", commitType, scope, description)
}
