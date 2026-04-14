package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"blooket-groq-csv/internal/template"
)

// Row is one exported question line.
type Row struct {
	Question string
	Answers  [4]string
	Correct  string
}

// WriteFile writes a Blooket-compatible CSV (banner + header + data).
func WriteFile(path string, col *template.ColumnMap, rows []Row, timeLimit string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// UTF-8 BOM matches official Blooket CSV / Excel-friendly imports.
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	w := csv.NewWriter(f)
	if runtime.GOOS == "windows" {
		w.UseCRLF = true
	}

	width := len(col.Headers)
	if width == 0 {
		return fmt.Errorf("no headers")
	}

	if col.BannerFirst {
		banner := make([]string, width)
		banner[0] = template.BannerCell()
		if err := w.Write(banner); err != nil {
			return err
		}
	}

	if err := w.Write(col.Headers); err != nil {
		return err
	}

	for i, r := range rows {
		rec := make([]string, width)
		if col.QuestionNum >= 0 && col.QuestionNum < width {
			rec[col.QuestionNum] = strconv.Itoa(i + 1)
		}
		if col.Question >= 0 && col.Question < width {
			rec[col.Question] = r.Question
		}
		for j := 0; j < 4; j++ {
			idx := col.Answer[j]
			if idx >= 0 && idx < width {
				rec[idx] = r.Answers[j]
			}
		}
		if col.TimeLimit >= 0 && col.TimeLimit < width {
			rec[col.TimeLimit] = timeLimit
		}
		if col.Correct >= 0 && col.Correct < width {
			rec[col.Correct] = r.Correct
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// ExportDir returns {exeDir}/exported.
func ExportDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(exe), "exported")
	return dir, nil
}

// DefaultOutPath builds a timestamped filename under ExportDir.
func DefaultOutPath() (string, error) {
	dir, err := ExportDir()
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("blooket_import_%s.csv", time.Now().Format("20060102_150405"))
	return filepath.Join(dir, name), nil
}
