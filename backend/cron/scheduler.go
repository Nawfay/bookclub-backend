package cron

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterCronJobs sets up all cron jobs for the application
func RegisterCronJobs(app core.App) {
	// Use the built-in app cron scheduler - this will show up in the admin UI
	app.Cron().MustAdd("process_notes", "*/5 * * * *", func() {
		log.Println("[Cron] Starting note processing...")
		ProcessUnprocessedNotes(app)
	})

	log.Println("[Cron] ✅ Registered cron job 'process_notes' - runs every 5 minutes")
}
