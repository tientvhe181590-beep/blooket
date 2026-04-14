package template

import (
	"fmt"
)

// ColumnMap maps logical fields to CSV column indices for the official Blooket import template.
type ColumnMap struct {
	Headers     []string
	BannerFirst bool // when true, writer emits row 0 with Blooket banner cell
	QuestionNum int  // "Question #" column; -1 if absent
	Question    int  // "Question Text"
	Answer      [4]int
	Correct     int
	TimeLimit   int
}

const bannerCell = "Blooket\nImport Template"

// BlooketColumnMap returns the layout matching Blooket_Spreadsheet_Import_Template CSV (26 columns).
func BlooketColumnMap() *ColumnMap {
	headers := make([]string, 26)
	headers[0] = "Question #"
	headers[1] = "Question Text"
	headers[2] = "Answer 1"
	headers[3] = "Answer 2"
	headers[4] = "Answer 3\n(Optional)"
	headers[5] = "Answer 4\n(Optional)"
	headers[6] = "Time Limit (sec)\n(Max: 300 seconds)"
	headers[7] = "Correct Answer(s)\n(Only include Answer #)"
	// headers[8:26] remain ""

	return &ColumnMap{
		Headers:     headers,
		BannerFirst: true,
		QuestionNum: 0,
		Question:    1,
		Answer:      [4]int{2, 3, 4, 5},
		TimeLimit:   6,
		Correct:     7,
	}
}

// BannerCell is the single non-empty cell in the first template row.
func BannerCell() string { return bannerCell }

// Validate checks required columns.
func (cm *ColumnMap) Validate() error {
	if cm.Question < 0 || cm.Answer[0] < 0 || cm.Answer[1] < 0 {
		return fmt.Errorf("internal error: bad column map")
	}
	if cm.Correct < 0 {
		return fmt.Errorf("internal error: missing correct-answer column")
	}
	return nil
}
