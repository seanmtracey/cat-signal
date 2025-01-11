package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Create a new Fiber app
	app := fiber.New()

	// Serve static files from the /static directory
	app.Static("/", "./static")

	// POST /update endpoint
	app.Post("/update", func(c *fiber.Ctx) error {
		// Log the request body
		body := c.Body()
		log.Printf("Request body: %s", string(body))

		// Respond with a success message
		return c.SendString("Update received")
	})

	// Start the server on port 3000
	log.Println("Server is running on http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
	
}
