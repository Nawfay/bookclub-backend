package cron

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/pocketbase/pocketbase/core"
)

// ProcessUnprocessedNotes handles the main cron job logic for processing notes
func ProcessUnprocessedNotes(app core.App) {
	log.Println("[Cron] 🔍 Checking for unprocessed notes...")

	// 1. Fetch unprocessed notes
	notes, err := app.FindRecordsByFilter(
		"notes",
		"processed = false",
		"-created",
		0,
		0,
	)
	if err != nil {
		log.Printf("[Cron] ❌ Error fetching notes: %v", err)
		return
	}

	if len(notes) == 0 {
		log.Println("[Cron] ✅ No unprocessed notes found")
		return
	}

	log.Printf("[Cron] 📝 Found %d unprocessed notes.", len(notes))

	// 2. Group notes by Book ID
	notesByBook := make(map[string][]*core.Record)
	for _, note := range notes {
		bookId := note.GetString("book")
		if bookId != "" {
			notesByBook[bookId] = append(notesByBook[bookId], note)
		}
	}

	// 3. Process each Book
	for bookId, bookNotes := range notesByBook {
		// A. Find the PDF file
		fileRecord, err := app.FindFirstRecordByFilter(
			"files",
			"book = {:bookId} && primaryFile = true",
			map[string]any{"bookId": bookId},
		)
		if err != nil {
			log.Printf("[Cron] No PDF found for book %s. Skipping.", bookId)
			continue
		}

		filePath := filepath.Join(app.DataDir(), "storage", fileRecord.Collection().Id, fileRecord.Id, fileRecord.GetString("filename"))

		// B. LOAD ENTIRE BOOK INTO MEMORY
		// Returns map[PageNumber]ContentString
		bookContent, err := extractAllPages(filePath)
		if err != nil {
			log.Printf("[Cron] Failed to read PDF %s: %v", bookId, err)
			continue
		}

		// C. Check all notes against the loaded book map
		for _, note := range bookNotes {
			targetText := cleanTextForSearch(note.GetString("bookText"))
			foundPage := 999

			// Iterate through our in-memory book map
			for pageNum, pageText := range bookContent {
				if strings.Contains(pageText, targetText) {
					foundPage = pageNum
					break // Stop searching once found
				}
			}

			// D. Update the record
			note.Set("processed", true)
			if foundPage < 999 {
				note.Set("page", foundPage)
				log.Printf("[Cron] MATCH: Note %s -> Page %d", note.Id, foundPage)
			} else {
				note.Set("page", foundPage)
				log.Printf("[Cron] FAIL: Could not find text for note %s", note.Id)
			}

			if err := app.Save(note); err != nil {
				log.Printf("[Cron] Database save failed for note %s: %v", note.Id, err)
			}
		}
	}
}

// extractAllPages reads the PDF once and returns a map of PageNum -> Text
func extractAllPages(path string) (map[int]string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	results := make(map[int]string)
	totalPage := r.NumPage()

	// Loop through every page and store it in the map
	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, _ := p.GetPlainText(nil)
		results[i] = cleanTextForSearch(text)
	}
	return results, nil
}

// cleanTextForSearch standardizes text to increase match rate
func cleanTextForSearch(text string) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "  ", " ") // collapse double spaces
	return strings.TrimSpace(text)
}
