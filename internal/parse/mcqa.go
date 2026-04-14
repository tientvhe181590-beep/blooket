package parse

import (
	"fmt"
	"regexp"
	"strings"
)

// Block is one multiple-choice item: stem and up to four options (A–D slots).
type Block struct {
	Question string
	Answers  [4]string
}

var optionLine = regexp.MustCompile(`(?m)^\s*([A-Da-d])[\.\)]\s*(.*)$`)

// ParseBlocks splits pasted text into blocks (blank-line separated) and extracts A–D options.
func ParseBlocks(text string) ([]Block, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("paste is empty")
	}

	rawBlocks := regexp.MustCompile(`\n\s*\n+`).Split(text, -1)
	var out []Block
	for i, raw := range rawBlocks {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		b, err := parseOneBlock(raw)
		if err != nil {
			return nil, fmt.Errorf("block %d: %w", i+1, err)
		}
		out = append(out, b)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no question blocks found")
	}
	return out, nil
}

func parseOneBlock(para string) (Block, error) {
	matches := optionLine.FindAllStringSubmatchIndex(para, -1)
	if len(matches) == 0 {
		return Block{}, fmt.Errorf("no A–D option lines (e.g. \"A. text\")")
	}

	firstOpt := matches[0][0]
	stem := strings.TrimSpace(para[:firstOpt])
	if stem == "" {
		stem = "(no question text)"
	}

letterText := map[byte]string{}
	for _, m := range matches {
		letter := strings.ToUpper(para[m[2]:m[3]])[0]
		optText := strings.TrimSpace(para[m[4]:m[5]])
		if optText == "" {
			optText = "(empty)"
		}
		if _, dup := letterText[letter]; dup {
			return Block{}, fmt.Errorf("duplicate option %c", letter)
		}
		letterText[letter] = optText
	}

	var b Block
	b.Question = stem
	for i := 0; i < 4; i++ {
		letter := byte('A' + i)
		if t, ok := letterText[letter]; ok {
			b.Answers[i] = t
		}
	}

	nonEmpty := 0
	for _, a := range b.Answers {
		if strings.TrimSpace(a) != "" {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		return Block{}, fmt.Errorf("need at least two answer choices (A and B)")
	}

	return b, nil
}

// labelOnlyLine matches a line that is only a question index label (no real stem text).
var labelOnlyLine = regexp.MustCompile(`(?i)^\s*(?:question\s*#?\s*\d+|q\.?\s*#?\s*\d+|c[aâu]*\s*\d+|\d+\s*[\).\]:])\s*$`)

// StripQuestionLabels removes leading lines that are only "Question 3", "Q1", "Câu 1", etc.
// Used as a fallback when the model does not return a cleaned question.
func StripQuestionLabels(stem string) string {
	stem = strings.ReplaceAll(stem, "\r\n", "\n")
	lines := strings.Split(stem, "\n")
	i := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			i++
			continue
		}
		if labelOnlyLine.MatchString(t) {
			i++
			continue
		}
		break
	}
	out := strings.TrimSpace(strings.Join(lines[i:], "\n"))
	if out == "" {
		return strings.TrimSpace(stem)
	}
	return out
}
