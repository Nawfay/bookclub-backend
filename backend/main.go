package main

import (
	"log"

	"sheikahslate/routes"

	"github.com/pocketbase/pocketbase"
)

func main() {
	app := pocketbase.New()


	routes.RegisterInviteRoute(app)
	routes.RegisterBookAdditionRoutes(app)

	log.Println("PocketBase backend starting...")
	log.Println("Admin UI available at: http://127.0.0.1:8090/_/")
	log.Println("API available at: http://127.0.0.1:8090/api/")

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
