package models

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}
type CommitSuggestion struct {
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
}
