package main

import (
	"log"
	"otsu-obliterator/internal/app"
)

func main() {
	log.Println("MAIN: Starting main function")

	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	log.Println("MAIN: About to call ForceMenuSetup")
	// Force menu setup before any GUI operations
	application.ForceMenuSetup()
	log.Println("MAIN: ForceMenuSetup completed")

	log.Println("MAIN: About to call Run")
	if err := application.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
	log.Println("MAIN: Run completed")
}
