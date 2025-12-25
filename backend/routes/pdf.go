package routes

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPDFRoute(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/book/{id}/read/{page}", func(e *core.RequestEvent) error {
			bookId := e.Request.PathValue("id")
			pageStr := e.Request.PathValue("page")

			// 1. Validate Page Number
			pageIndex, err := strconv.Atoi(pageStr)
			if err != nil || pageIndex < 1 {
				return e.BadRequestError("Invalid page number", err)
			}

			// 2. Find the Primary File for this Book
			// We query the 'files' collection where 'book' matches and 'primaryFile' is true
			record, err := app.FindFirstRecordByFilter(
				"files",
				"book = {:bookId} && primaryFile = true",
				map[string]any{"bookId": bookId},
			)

			if err != nil {
				return e.NotFoundError("Book file not found", err)
			}

			// 3. Construct the Filesystem Path
			// PocketBase stores files in: /pb_data/storage/{collectionId}/{recordId}/{filename}
			filename := record.GetString("filename")
			collectionId := record.Collection().Id
			recordId := record.Id

			// Get the data directory path
			filePath := filepath.Join(app.DataDir(), "storage", collectionId, recordId, filename)

			// 4. Extract Text from PDF (simplified for now - you'll need to add PDF library)
			content, err := extractPageText(filePath, pageIndex)
			if err != nil {
				// If page is out of bounds, return empty content or specific error
				return e.InternalServerError("Failed to extract PDF content", err)
			}

			// 5. Return JSON in the format your frontend expects (array of strings/paragraphs)
			// We split by newline to simulate paragraphs
			return e.JSON(http.StatusOK, map[string]any{
				"page":       pageIndex,
				"content":    splitIntoParagraphs(content),
				"contentRaw": content,
			})
		})

		return se.Next()
	})
}

// Helper: Extract text from a specific page using ledongthuc/pdf
func extractPageText(path string, targetPage int) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	totalPage := r.NumPage()
	if targetPage > totalPage {
		return "", fmt.Errorf("page %d exceeds total pages (%d)", targetPage, totalPage)
	}

	// ledongthuc/pdf reader allows reading specific pages
	// Note: It's 1-indexed in the GetPage function usually, but let's check reader.
	// Reader.Page(i) returns the page.
	p := r.Page(targetPage)

	if p.V.IsNull() {
		return "", nil
	}

	text, err := p.GetPlainText(nil)
	if err != nil {
		return "", err
	}

	cleanText := strings.ReplaceAll(text, "\t", " ")

	return cleanText, nil
}

// Helper: Split raw text into "paragraphs" for the UI
func splitIntoParagraphs(text string) []string {
	// Split by double newlines for paragraph breaks
	var paragraphs []string

	// Split by double newlines first
	parts := strings.Split(text, "\n\n")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			paragraphs = append(paragraphs, part)
		}
	}

	// If no double newlines found, split by single newlines but group them
	if len(paragraphs) <= 1 && strings.Contains(text, "\n") {
		paragraphs = []string{}
		lines := strings.Split(text, "\n")
		currentParagraph := ""

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				if currentParagraph != "" {
					paragraphs = append(paragraphs, currentParagraph)
					currentParagraph = ""
				}
			} else {
				if currentParagraph != "" {
					currentParagraph += " "
				}
				currentParagraph += line
			}
		}

		// Add the last paragraph if it exists
		if currentParagraph != "" {
			paragraphs = append(paragraphs, currentParagraph)
		}
	}

	// If still only one paragraph or very long paragraphs, break them down further
	if len(paragraphs) == 1 || hasLongParagraphs(paragraphs) {
		paragraphs = breakDownLongParagraphs(paragraphs)
	}

	// If still no paragraphs, return the original text as one paragraph
	if len(paragraphs) == 0 {
		paragraphs = append(paragraphs, strings.TrimSpace(text))
	}

	return paragraphs
}

// Helper: Check if any paragraph is too long (more than 500 characters)
func hasLongParagraphs(paragraphs []string) bool {
	for _, p := range paragraphs {
		if len(p) > 500 {
			return true
		}
	}
	return false
}

// Helper: Break down long paragraphs into smaller chunks
func breakDownLongParagraphs(paragraphs []string) []string {
	var result []string

	for _, paragraph := range paragraphs {
		if len(paragraph) <= 500 {
			result = append(result, paragraph)
			continue
		}

		// Split long paragraphs by sentence endings
		sentences := strings.FieldsFunc(paragraph, func(c rune) bool {
			return c == '.' || c == '!' || c == '?'
		})

		currentChunk := ""
		for i, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}

			// Add the punctuation back (except for the last sentence)
			if i < len(sentences)-1 {
				sentence += "."
			}

			// If adding this sentence would make the chunk too long, start a new chunk
			if len(currentChunk) > 0 && len(currentChunk)+len(sentence)+1 > 400 {
				result = append(result, strings.TrimSpace(currentChunk))
				currentChunk = sentence
			} else {
				if currentChunk != "" {
					currentChunk += " "
				}
				currentChunk += sentence
			}
		}

		// Add the last chunk if it exists
		if currentChunk != "" {
			result = append(result, strings.TrimSpace(currentChunk))
		}
	}

	return result
}
