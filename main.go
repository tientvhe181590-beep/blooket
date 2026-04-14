package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"blooket-groq-csv/internal/export"
	"blooket-groq-csv/internal/groq"
	"blooket-groq-csv/internal/parse"
	"blooket-groq-csv/internal/template"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const prefAPIKey = "api_key"

func main() {
	a := app.NewWithID("com.blooket.csvexporter")
	w := a.NewWindow("Blooket CSV Exporter")
	w.Resize(fyne.NewSize(880, 640))

	prefs := a.Preferences()
	if prefs.String(prefAPIKey) == "" && prefs.String("groq_api_key") != "" {
		prefs.SetString(prefAPIKey, prefs.String("groq_api_key"))
		prefs.RemoveValue("groq_api_key")
	}

	qEntry := widget.NewMultiLineEntry()
	qEntry.Wrapping = fyne.TextWrapWord
	qEntry.SetPlaceHolder("Paste questions here. Separate blocks with a blank line.\n\nExample:\nWhat is 2+2?\nA. 3\nB. 4\nC. 5\nD. 6")

	keyEntry := widget.NewPasswordEntry()
	keyEntry.SetPlaceHolder("Key (hidden)")

	keyStatus := widget.NewLabel("")
	refreshKeyStatus := func() {
		if strings.TrimSpace(prefs.String(prefAPIKey)) != "" {
			keyStatus.SetText("A saved key is stored. Paste a new one and click Save to replace it.")
		} else {
			keyStatus.SetText("No key saved yet.")
		}
	}
	refreshKeyStatus()

	saveKeyBtn := widget.NewButton("Save key", func() {
		k := strings.TrimSpace(keyEntry.Text)
		if k == "" {
			dialog.ShowInformation("Key", "Enter a key before saving, or use Clear saved key.", w)
			return
		}
		prefs.SetString(prefAPIKey, k)
		keyEntry.SetText("")
		refreshKeyStatus()
		dialog.ShowInformation("Key", "Saved.", w)
	})

	clearKeyBtn := widget.NewButton("Clear saved key", func() {
		prefs.RemoveValue(prefAPIKey)
		keyEntry.SetText("")
		refreshKeyStatus()
	})

	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Seconds per question (e.g. 20)")
	timeEntry.Text = "20"

	status := widget.NewLabel("Ready.")

	scroll := container.NewScroll(qEntry)
	scroll.SetMinSize(fyne.NewSize(760, 360))

	exportBtn := widget.NewButton("Export CSV", nil)
	exportBtn.OnTapped = func() {
		exportBtn.Disable()
		status.SetText("Working…")

		go func() {
			defer fyne.Do(func() { exportBtn.Enable() })

			col := template.BlooketColumnMap()
			if err := col.Validate(); err != nil {
				fyne.Do(func() {
					status.SetText("Error.")
					dialog.ShowError(err, w)
				})
				return
			}

			blocks, err := parse.ParseBlocks(qEntry.Text)
			if err != nil {
				fyne.Do(func() {
					status.SetText("Parse error.")
					dialog.ShowError(err, w)
				})
				return
			}

			sec := strings.TrimSpace(timeEntry.Text)
			if sec == "" {
				fyne.Do(func() {
					status.SetText("Error.")
					dialog.ShowError(fmt.Errorf("enter time limit seconds"), w)
				})
				return
			}
			if _, err := strconv.Atoi(sec); err != nil {
				fyne.Do(func() {
					status.SetText("Error.")
					dialog.ShowError(fmt.Errorf("time limit must be an integer"), w)
				})
				return
			}

			apiKey := strings.TrimSpace(keyEntry.Text)
			if apiKey == "" {
				apiKey = prefs.String(prefAPIKey)
			}
			client := groq.Client{
				APIKey: strings.TrimSpace(apiKey),
			}

			ctx := context.Background()
			rows := make([]export.Row, 0, len(blocks))
			for i, b := range blocks {
				n := i + 1
				fyne.Do(func() {
					status.SetText(fmt.Sprintf("Question %d / %d", n, len(blocks)))
				})

				res, err := client.Infer(ctx, b.Question, b.Answers)
				if err != nil {
					fyne.Do(func() {
						status.SetText("Error.")
						dialog.ShowError(fmt.Errorf("question %d: %w", n, err), w)
					})
					return
				}
				rows = append(rows, export.Row{
					Question: res.Question,
					Answers:  b.Answers,
					Correct:  res.Correct,
				})
			}

			outPath, err := export.DefaultOutPath()
			if err != nil {
				fyne.Do(func() {
					status.SetText("Error.")
					dialog.ShowError(err, w)
				})
				return
			}
			if err := export.WriteFile(outPath, col, rows, sec); err != nil {
				fyne.Do(func() {
					status.SetText("Error.")
					dialog.ShowError(err, w)
				})
				return
			}

			fyne.Do(func() {
				status.SetText("Done.")
				dialog.ShowInformation("Exported", outPath, w)
			})
		}()
	}

	top := container.NewVBox(
		container.NewGridWithColumns(2,
			widget.NewLabel("Key"),
			keyEntry,
		),
		container.NewHBox(saveKeyBtn, clearKeyBtn),
		keyStatus,
		container.NewGridWithColumns(2,
			widget.NewLabel("Time limit (seconds, all questions)"),
			timeEntry,
		),
		widget.NewSeparator(),
		widget.NewLabel("Questions & answers (paste)"),
	)

	bottom := container.NewVBox(
		status,
		exportBtn,
	)

	w.SetContent(container.NewBorder(top, bottom, nil, nil, scroll))
	w.ShowAndRun()
}
