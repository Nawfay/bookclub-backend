package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func RegisterNotesRoute(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// POST /api/import-apple-books - Import Apple Books notes from email webhook
		se.Router.POST("/notes/import-apple-books", func(e *core.RequestEvent) error {
			// ---------------------------------------------------------
			// 1. EXTRACT DATA FROM WEBHOOK
			// ---------------------------------------------------------
			htmlContent := e.Request.FormValue("body-html")
			senderEmail := e.Request.FormValue("sender") // e.g. "dishit79@icloud.com"

			if htmlContent == "" {
				return e.BadRequestError("No 'body-html' field found", nil)
			}

			// ---------------------------------------------------------
			// 2. RESOLVE USER (Required by your 'notes' schema)
			// ---------------------------------------------------------
			// We try to find the user in PocketBase by the email sending the webhook
			userRecord, err := app.FindFirstRecordByFilter("users", "email={:email}", map[string]any{
				"email": senderEmail,
			})
			if err != nil {
				// Option A: Fail if user not found
				return e.BadRequestError("Could not find user with email: "+senderEmail, err)
			}

			// ---------------------------------------------------------
			// 3. PARSE HTML
			// ---------------------------------------------------------
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
			if err != nil {
				return e.BadRequestError("Failed to parse HTML", err)
			}

			// Extract Book Title & Author
			title := strings.TrimSpace(doc.Find(".booktitle").Text())
			// author := strings.TrimSpace(doc.Find(".booktitle + h2").Text())

			// Fallback if HTML parsing fails (sometimes Apple changes classes)
			if title == "" {
				subject := e.Request.FormValue("subject") // "Notes from "The Silent Patient"..."
				if strings.Contains(subject, `“`) && strings.Contains(subject, `”`) {
					parts := strings.Split(subject, `“`)
					if len(parts) > 1 {
						title = strings.Split(parts[1], `”`)[0]
					}
				}
			}

			if title == "" {
				return e.BadRequestError("Could not determine book title", nil)
			}

			// ---------------------------------------------------------
			// 4. FIND OR CREATE BOOK
			// ---------------------------------------------------------

			bookRecord, err := app.FindFirstRecordByFilter("books", "title={:title}", map[string]any{
				"title": title,
			})
			if err != nil {

				return e.BadRequestError("Book Does Not Exist", err)
			}

			// ---------------------------------------------------------
			// 5. PROCESS NOTES
			// ---------------------------------------------------------
			notesCollection, err := app.FindCollectionByNameOrId("notes")
			if err != nil {
				return e.InternalServerError("Notes collection not found", err)
			}

			var importCount int

			doc.Find(".annotation").Each(func(i int, s *goquery.Selection) {
				// Scrape the data
				// Note: Your schema has 'page' as a Number (#).
				// Apple Books gives "Chapter One" (text).
				// We cannot safely map Text -> Number, so we skip 'page' for now.

				quote := strings.TrimSpace(s.Find(".annotationrepresentativetext").Text()) // -> bookText
				note := strings.TrimSpace(s.Find(".annotationnote").Text())                // -> note

				// Extract raw date string: "December 1, 2025"
				rawDate := strings.TrimSpace(s.Find(".annotationdate").Text())

				// Skip empty entries
				if quote == "" && note == "" {
					return
				}

				// Check for duplicates to prevent spamming
				// Modern syntax: Find records by filter
				existingRecords, _ := app.FindRecordsByFilter(
					"notes",
					"book = {:book} && bookText = {:text} && user = {:user}",
					"-created",
					1,
					0,
					map[string]any{
						"book": bookRecord.Id,
						"text": quote,
						"user": userRecord.Id,
					},
				)

				if len(existingRecords) == 0 {
					// Create new Note
					newNote := core.NewRecord(notesCollection)

					newNote.Set("book", bookRecord.Id)
					newNote.Set("user", userRecord.Id)
					newNote.Set("bookText", quote) // Mapped to your 'bookText' field
					newNote.Set("note", note)      // Mapped to your 'note' field
					newNote.Set("processed", false)

					if rawDate != "" {
						parsedTime, err := time.Parse("January 2, 2006", rawDate)
						if err == nil {
							// Convert standard Go time to PocketBase DateTime type
							pbDate, _ := types.ParseDateTime(parsedTime)
							newNote.Set("created", pbDate)
						}
					}

					if err := app.Save(newNote); err == nil {
						importCount++
					}
				}
			})

			return e.JSON(http.StatusOK, map[string]any{
				"status":   "success",
				"book":     title,
				"imported": importCount,
				"user":     userRecord.GetString("email"),
			})
		})

		return se.Next()
	})
}
