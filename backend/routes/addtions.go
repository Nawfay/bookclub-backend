package routes

import (
	"io"
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func RegisterBookAdditionRoutes(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// POST /books/add/api - Add a new book with cover image download from URL
		se.Router.POST("/books/add/api", func(e *core.RequestEvent) error {
			// 1. Define the expected structure of the incoming JSON
			data := struct {
				Title    string `json:"title"`
				Author   string `json:"author"`
				Pages    int    `json:"pages"`
				CoverUrl string `json:"coverUrl"`
			}{}

			// 2. Parse the body
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Invalid data format", err)
			}

			// 3. Find the 'books' collection
			collection, err := app.FindCollectionByNameOrId("books")
			if err != nil {
				return e.InternalServerError("Books collection not found", err)
			}

			// 4. Initialize the record
			record := core.NewRecord(collection)
			record.Set("title", data.Title)
			record.Set("author", data.Author)
			record.Set("totalPages", data.Pages)
			record.Set("coverImageUrl", data.CoverUrl) // Save the URL string just in case
			record.Set("status", "planned")

			println(data.CoverUrl)
			
			if data.CoverUrl == "" {
				data.CoverUrl = "https://openlibrary.org/images/icons/avatar_book-sm.png"
			}

			// 5. Download and attach the cover image (if URL exists)
			if data.CoverUrl != "" {
				resp, err := http.Get(data.CoverUrl)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()

					// Read the image bytes into memory
					imgBytes, err := io.ReadAll(resp.Body)
					if err == nil {
						// Create a PocketBase file from the bytes
						// We name it "cover.jpg", PB will auto-generate a unique suffix if needed
						f, err := filesystem.NewFileFromBytes(imgBytes, "cover.jpg")
						if err == nil {
							record.Set("cover", f)
						}
					}
				}
			}

			// 6. Save the record to the database
			if err := app.Save(record); err != nil {
				return e.InternalServerError("Failed to save book record", err)
			}

			return e.JSON(http.StatusOK, record)
		})

		// POST /books/add/manual - Add a new book with manual form data and file upload
		se.Router.POST("/books/add/manual", func(e *core.RequestEvent) error {
			// Parse multipart form data
			if err := e.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB max
				return e.BadRequestError("Failed to parse form data", err)
			}

			// Get form values
			title := e.Request.FormValue("title")
			author := e.Request.FormValue("author")
			pagesStr := e.Request.FormValue("pages")
			coverUrl := e.Request.FormValue("coverUrl")

			// Validate required fields
			if title == "" || author == "" {
				return e.BadRequestError("Title and Author are required", nil)
			}

			// Parse pages
			pages := 0
			if pagesStr != "" {
				if parsedPages, err := strconv.Atoi(pagesStr); err == nil {
					pages = parsedPages
				}
			}

			// Find the 'books' collection
			collection, err := app.FindCollectionByNameOrId("books")
			if err != nil {
				return e.InternalServerError("Books collection not found", err)
			}

			// Create the new record
			record := core.NewRecord(collection)
			record.Set("title", title)
			record.Set("author", author)
			record.Set("totalPages", pages)
			record.Set("coverImageUrl", coverUrl)
			record.Set("status", "planned")

			// Handle file upload (cover image)
			file, fileHeader, err := e.Request.FormFile("cover")
			if err == nil && file != nil {
				defer file.Close()

				// Read the uploaded file
				fileBytes, err := io.ReadAll(file)
				if err == nil && len(fileBytes) > 0 {
					// Create a PocketBase file from the uploaded bytes
					// Use the original filename or default to cover.jpg
					filename := fileHeader.Filename
					if filename == "" {
						filename = "cover.jpg"
					}

					f, err := filesystem.NewFileFromBytes(fileBytes, filename)
					if err == nil {
						record.Set("cover", f)
					}
				}
			} else if coverUrl != "" {
				// Fallback: try to download from URL if no file was uploaded
				resp, err := http.Get(coverUrl)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()

					imgBytes, err := io.ReadAll(resp.Body)
					if err == nil && len(imgBytes) > 0 {
						f, err := filesystem.NewFileFromBytes(imgBytes, "cover.jpg")
						if err == nil {
							record.Set("cover", f)
						}
					}
				}
			}

			// Save to database
			if err := app.Save(record); err != nil {
				return e.InternalServerError("Failed to save book", err)
			}

			return e.JSON(http.StatusOK, record)
		})

		return se.Next()
	})
}
