package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"blooket-groq-csv/internal/parse"
)

const defaultEndpoint = "https://api.groq.com/openai/v1/chat/completions"

// DefaultModelChain is tried in order until one request succeeds.
var DefaultModelChain = []string{
	"llama-3.3-70b-versatile",
	"llama-3.1-8b-instant",
	"llama-3.1-70b-versatile",
}

// Client calls the chat completions HTTP API.
type Client struct {
	APIKey     string
	ModelChain []string // if empty, DefaultModelChain is used
	Endpoint   string
	HTTP       *http.Client
}

// InferResult is the model output for one MC block.
type InferResult struct {
	Correct  string // "1" or "1,3"
	Question string // stem only, no "Question 3" / "Q1" / "Câu 1" prefixes
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type inferPayload struct {
	Correct  string `json:"correct"`
	Question string `json:"question"`
}

var fenceRE = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

// Infer asks for the correct option index(es) and a cleaned question stem for CSV export.
func (c *Client) Infer(ctx context.Context, question string, answers [4]string) (*InferResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("missing API key")
	}
	chain := c.ModelChain
	if len(chain) == 0 {
		chain = DefaultModelChain
	}

	var lastErr error
	for _, model := range chain {
		res, err := c.inferWithModel(ctx, model, question, answers)
		if err == nil {
			return res, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no models configured")
	}
	return nil, fmt.Errorf("%w (tried %d model(s))", lastErr, len(chain))
}

func (c *Client) inferWithModel(ctx context.Context, model, question string, answers [4]string) (*InferResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("empty model name")
	}
	endpoint := strings.TrimSpace(c.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}

	var opts strings.Builder
	for i := 0; i < 4; i++ {
		a := strings.TrimSpace(answers[i])
		if a == "" {
			continue
		}
		fmt.Fprintf(&opts, "%d: %s\n", i+1, a)
	}

	rawStem := strings.TrimSpace(question)
	prompt := fmt.Sprintf(`You are grading a multiple-choice item.

The block below may include redundant label lines (e.g. "Question 3", "Q1", "Câu 1") before the real question. For the JSON field "question", output ONLY the actual question text a student reads — no "Question N", "Q1", "Câu 1", numbering prefixes, or duplicate labels. Preserve the real wording and punctuation (including quotes inside the question).

Pick the single best correct option index (1-4) for the numbered options. If multiple answers are clearly correct, use a comma-separated list with no spaces (e.g. "1,3").

Question block:
%s

Options (index: text):
%s

Respond with ONLY valid JSON, no markdown, in this exact shape:
{"correct":"2","question":"..."}`,
		rawStem, opts.String())

	body, err := json.Marshal(chatRequest{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if cr.Error != nil && cr.Error.Message != "" {
		return nil, fmt.Errorf("%s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in response")
	}
	content := strings.TrimSpace(cr.Choices[0].Message.Content)
	content = extractJSON(content)

	var payload inferPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return nil, fmt.Errorf("parse model JSON %q: %w", content, err)
	}
	corr := strings.TrimSpace(payload.Correct)
	if corr == "" {
		return nil, fmt.Errorf("empty correct field in model output")
	}
	if err := validateCorrect(corr, answers); err != nil {
		return nil, err
	}

	cleanQ := strings.TrimSpace(payload.Question)
	if cleanQ == "" {
		cleanQ = parse.StripQuestionLabels(rawStem)
	}

	return &InferResult{Correct: corr, Question: cleanQ}, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if m := fenceRE.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return s
}

func validateCorrect(corr string, answers [4]string) error {
	parts := strings.Split(corr, ",")
	if len(parts) == 0 {
		return fmt.Errorf("invalid correct %q", corr)
	}
	seen := map[int]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return fmt.Errorf("invalid correct %q", corr)
		}
		idx, err := strconv.Atoi(p)
		if err != nil || idx < 1 || idx > 4 {
			return fmt.Errorf("correct index out of range in %q", corr)
		}
		if strings.TrimSpace(answers[idx-1]) == "" {
			return fmt.Errorf("correct index %d has no answer text", idx)
		}
		if seen[idx] {
			return fmt.Errorf("duplicate index in %q", corr)
		}
		seen[idx] = true
	}
	return nil
}
