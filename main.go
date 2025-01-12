package main

import (
	"fmt"
	"os"
	"os/signal"
	"bufio"
	"context"
	"flag"
	"strconv"
	"math"
	"strings"
	"sync"
	"syscall"
	"time"
	ledColor "image/color"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/gofiber/fiber/v2"
	"github.com/mcuadros/go-rpi-ws281x"
	robot "github.com/tlkamp/litter-api/v2/pkg/client"
)

type Payload struct {
	Data []string `json:"data"`
}

// Global variables
var (
	USERNAME       string
	PASSWORD       string
	CHECK_INTERVAL int
	CHECK_COUNTER  int

	LOGIN_RETRY_DELAY float64 = 5.0
	API               *robot.Client
	CTX               context.Context

	LEDColor       ledColor.RGBA = ledColor.RGBA{0, 0, 255, 255} // Default LED color (blue)
	mu             sync.Mutex                                   // Mutex to protect LEDColor during updates
	LED_DIRECTION  = []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	LED_VALUES     = []int{21, 42, 63, 84, 105, 126, 147, 168, 189, 210, 231, 252}
	pin            = flag.Int("gpio-pin", 18, "GPIO pin")
	width          = flag.Int("width", 12, "LED matrix width")
	height         = flag.Int("height", 1, "LED matrix height")
	brightness     = flag.Int("brightness", 64, "Brightness (0-255)")
	statusColors   []ledColor.RGBA
)

// Map of color names to RGB values
var colorNameToRGB = map[string]ledColor.RGBA{
	"red":    {255, 0, 0, 255},
	"green":  {0, 255, 0, 255},
	"blue":   {0, 0, 255, 255},
	"yellow": {255, 255, 0, 255},
	"orange": {255, 165, 0, 255},
	"black":  {0, 0, 0, 255},
	"white":  {255, 255, 255, 255},
}

// Helper function to parse colors from a file
func loadStatusColors(filename string) ([]ledColor.RGBA, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var colors []ledColor.RGBA
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text())) // Normalize color name
		if rgb, ok := colorNameToRGB[line]; ok {
			colors = append(colors, rgb)
		} else {
			color.Red("Unknown color name in status-colors.txt: %s", line)
			colors = append(colors, colorNameToRGB["black"]) // Default to black for unknown colors
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return colors, nil
}

func setStatusColors(){
	
	var err error
	statusColors, err = loadStatusColors("status-colors.txt")
	if err != nil {
		color.Red("Failed to load status colors. Using default colors.")
		
		var colors []ledColor.RGBA

		for _ = range 10 {
			
			colors = append(colors, colorNameToRGB["black"])

		}
		
		statusColors = colors

	}

}

// Set LED color safely
func setLEDColor(color ledColor.RGBA) {
	mu.Lock()
	LEDColor = color
	mu.Unlock()
}

// Animate LEDs
func animate(c *ws281x.Canvas) {
	bounds := c.Bounds()

	for {
		mu.Lock()
		baseColor := LEDColor // Get the current base color safely
		mu.Unlock()

		for idx := range LED_VALUES {
			if LED_DIRECTION[idx] == 1 {
				LED_VALUES[idx] += int(255 / 21)
			} else {
				LED_VALUES[idx] -= int(255 / 21)
			}

			if LED_VALUES[idx] >= 255 {
				LED_VALUES[idx] = 255
				LED_DIRECTION[idx] = -1
			} else if LED_VALUES[idx] <= 0 {
				LED_VALUES[idx] = 0
				LED_DIRECTION[idx] = 1
			}
		}

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				idx := x
				if idx < len(LED_VALUES) {
					brightness := float64(LED_VALUES[idx]) / 255.0
					c.Set(x, y, scaleColor(baseColor, brightness))
				}
			}
		}
		c.Render()
		time.Sleep(100 * time.Millisecond)
	}
}

// Helper function to scale RGB color by brightness
func scaleColor(col ledColor.RGBA, brightness float64) ledColor.RGBA {
	return ledColor.RGBA{
		R: uint8(float64(col.R) * brightness),
		G: uint8(float64(col.G) * brightness),
		B: uint8(float64(col.B) * brightness),
		A: col.A,
	}
}

// Robot logic
func loginToServiceAndSetContext() {
	API = robot.New(USERNAME, PASSWORD)
	CTX = context.Background()

	color.Cyan(" > Logging in to service and setting context for requests.")

	for {
		err := API.Login(CTX)
		if err != nil {
			LOGIN_RETRY_DELAY = math.Min(math.Round(LOGIN_RETRY_DELAY*1.25), 300)
			color.Red("🚫 Could not login to robot.")

			setLEDColor(ledColor.RGBA{255, 255, 0, 255}) // Yellow

			color.Red(" > " + fmt.Sprintf("%s", err.Error()))
			color.Red(fmt.Sprintf(" > [ %s ] Waiting %f seconds before retrying login...\n\n", getTimeString(), LOGIN_RETRY_DELAY))
			time.Sleep(time.Second * time.Duration(LOGIN_RETRY_DELAY))
		} else {
			color.Green("✅ Logged-in to robot service.")
			setLEDColor(ledColor.RGBA{0, 0, 255, 255}) // Blue
			LOGIN_RETRY_DELAY = 5.0
			break
		}
	}
}

func checkStatusOfRobot() {

	setStatusColors()

	totalTimeSinceLastLogin := time.Second * time.Duration(CHECK_COUNTER*CHECK_INTERVAL)

	if totalTimeSinceLastLogin >= time.Minute*10 {
		color.Yellow(" > Session is too old. Re-authenticating...")
		loginToServiceAndSetContext()
		CHECK_COUNTER = 0
	}

	color.Cyan(fmt.Sprintf("\n > [ %s ] Getting status of litter robots...\n", getTimeString()))

	if err := API.FetchRobots(CTX); err != nil {
		color.Yellow(fmt.Sprintf("⚠️ [ %s ] Could not get robot details. Retrying in %d seconds...", getTimeString(), CHECK_INTERVAL))
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	} else {
		registeredRobots := API.Robots()
		for idx, r := range registeredRobots {
			color.Magenta(fmt.Sprintf("\n>>>>>>>>>>>>>>>>>>>>\n>> Robot %d of %d\n>>>>>>>>>>>>>>>>>>>>\n\n", idx+1, len(registeredRobots)))
			color.Magenta("\tRobot ID:")
			color.White(fmt.Sprintf("\t%s\n\n", r.LitterRobotID))

			color.Magenta("\tRobot Name:")
			color.White(fmt.Sprintf("\t%s\n\n", r.Name))

			unitStatus := int(r.UnitStatus)
			statusText := mapUnitStatusToString(float64(unitStatus))
			color.Magenta("\tRobot Status:\n")
			color.White(fmt.Sprintf("\t%s\n\n", statusText))

			// Set LEDs to color based on unit status
			if unitStatus < len(statusColors) {
				setLEDColor(statusColors[unitStatus])
			} else {
				setLEDColor(colorNameToRGB["black"]) // Default to black for out-of-range statuses
			}
		}
	}

	CHECK_COUNTER++
	color.Cyan(fmt.Sprintf(" > [ %s ] Waiting %d seconds before checking again...\n\n", getTimeString(), CHECK_INTERVAL))
}

// Utility functions
func mapUnitStatusToString(status float64) string {
	statusMap := map[float64]string{
		0:  "Ready",
		1:  "Clean Cycle in Progress",
		2:  "Clean Cycle Complete",
		3:  "Cat Sensor Fault",
		4:  "Drawer full; Will still cycle.",
		5:  "Drawer full; Will still cycle.",
		6:  "Cat Sensor Timing.",
		7:  "Cat Sensor Interrupt.",
		8:  "Bonnet Removed.",
		9:  "Paused.",
		10: "Off.",
		11: "Drawer full; Will not cycle.",
		12: "Drawer full; Will not cycle.",
	}
	if description, ok := statusMap[status]; ok {
		return description
	}
	return "Unknown Status"
}

func handleShutdown(c *ws281x.Canvas) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		color.Red("\nShutting down gracefully...")
		setLEDColor(colorNameToRGB["black"]) // Black
		c.Render()
		time.Sleep(100 * time.Millisecond)
		c.Close()
		os.Exit(0)
	}()
}

func startServer(){

	app := fiber.New()

	// Serve static files from the /static directory
	app.Static("/", "./static")

	// POST /update endpoint
	app.Post("/update", func(c *fiber.Ctx) error {
		// Parse the request body into the Payload struct
		var payload Payload
		if err := c.BodyParser(&payload); err != nil {
			fmt.Printf("Error parsing request body: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid JSON payload")
		}

		// Open the file for writing
		file, err := os.Create("status-colors.txt")
		if err != nil {
			fmt.Printf("Error creating file: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to write to file")
		}
		defer file.Close()

		// Write each color value to the file
		for _, color := range payload.Data {
			if _, err := file.WriteString(color + "\n"); err != nil {
				fmt.Printf("Error writing to file: %v", err)
				return c.Status(fiber.StatusInternalServerError).SendString("Failed to write to file")
			}
		}

		fmt.Println("Colors successfully written to status-colors.txt")

		c.Status(200)
		return c.Send(nil)

	})

	app.Get("/colors", func(c *fiber.Ctx) error {
		
		c.Status(200)
		return c.SendFile("status-colors.txt")

	})

	app.Listen(":80")

}

func main() {

	dotenvErr := godotenv.Load()
	if dotenvErr != nil {
		color.Cyan(">  No .env file detected. Defaulting to system env-vars instead.")
	}

	USERNAME = os.Getenv("ROBOT_EMAIL")
	PASSWORD = os.Getenv("ROBOT_PASS")

	interval := os.Getenv("CHECK_INTERVAL")
	if interval == "" {
		CHECK_INTERVAL = 60
	} else {
		value, convErr := strconv.Atoi(interval)
		if convErr != nil {
			CHECK_INTERVAL = 60
		} else {
			CHECK_INTERVAL = value
		}
	}

	config := ws281x.DefaultConfig
	config.Brightness = *brightness
	config.Pin = *pin

	c, err := ws281x.NewCanvas(*width, *height, &config)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = c.Initialize()
	if err != nil {
		panic(err)
	}

	handleShutdown(c)
	go animate(c)

	go startServer()

	loginToServiceAndSetContext()

	for {
		checkStatusOfRobot()
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	}
}

func getTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
