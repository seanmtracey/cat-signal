package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

type Payload struct {
	Data []string `json:"data"`
}

func main() {
	// Create a new Fiber app
	app := fiber.New()

	// Serve static files from the /static directory
	app.Static("/", "./static")

	// POST /update endpoint
	app.Post("/update", func(c *fiber.Ctx) error {
		// Parse the request body into the Payload struct
		var payload Payload
		if err := c.BodyParser(&payload); err != nil {
			log.Printf("Error parsing request body: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON payload")
		}

		// Open the file for writing
		file, err := os.Create("status-colors.txt")
		if err != nil {
			log.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to write to file")
		}
		defer file.Close()

		// Write each color value to the file
		for _, color := range payload.Data {
			if _, err := file.WriteString(color + "\n"); err != nil {
				log.Printf("Error writing to file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to write to file")
			}
		}

		log.Println("Colors successfully written to status-colors.txt")

		c.Status(200)
		return c.Send(nil)

	})

	app.Get("/colors", func(c *fiber.Ctx) error {
		
		c.Status(200)
		return c.SendFile("status-colors.txt")

	})

	// Start the server on port 3000
	log.Println("Server is running on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))

}
