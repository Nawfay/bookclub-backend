package main

import (
	"log"

	"sheikahslate/cron"
	"sheikahslate/routes"

	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()

	// Configure server to bind to all interfaces on port 8768
	app.RootCmd.SetArgs([]string{"serve", "--http=0.0.0.0:8090"})

	routes.RegisterInviteRoute(app)
	routes.RegisterBookAdditionRoutes(app)
	routes.RegisterNotesRoute(app)
	routes.RegisterPDFRoute(app)

	// Register cron jobs
	cron.RegisterCronJobs(app)

	log.Println("PocketBase backend starting...")
	log.Println("Admin UI available at: http://0.0.0.0:8768/_/")
	log.Println("API available at: http://0.0.0.0:8768/api/")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
