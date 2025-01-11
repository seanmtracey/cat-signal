package main

import (
	"context"
	"flag"
	"fmt"
	ledColor "image/color" // Aliasing image/color to ledColor
	"math"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/mcuadros/go-rpi-ws281x"
	robot "github.com/tlkamp/litter-api/v2/pkg/client"
)

// Global variables
var (
	USERNAME       string
	PASSWORD       string
	CHECK_INTERVAL int
	CHECK_COUNTER  int

	LOGIN_RETRY_DELAY float64 = 5.0
	API               *robot.Client
	CTX               context.Context

	LEDColor       ledColor.RGBA = ledColor.RGBA{255, 0, 0, 255} // Default LED color (red)
	mu             sync.Mutex                                   // Mutex to protect LEDColor during updates
	LED_DIRECTION  = []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	LED_VALUES     = []int{21, 42, 63, 84, 105, 126, 147, 168, 189, 210, 231, 252}
	pin            = flag.Int("gpio-pin", 18, "GPIO pin")
	width          = flag.Int("width", 12, "LED matrix width")
	height         = flag.Int("height", 1, "LED matrix height")
	brightness     = flag.Int("brightness", 64, "Brightness (0-255)")
)

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
				LED_VALUES[idx] += int(math.Floor(255 / 21))
			} else {
				LED_VALUES[idx] -= int(math.Floor(255 / 21))
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
	totalTimeSinceLastLogin := time.Second * time.Duration(CHECK_COUNTER*CHECK_INTERVAL)

	if totalTimeSinceLastLogin >= time.Minute*10 {
		color.Yellow(" > Session is too old. Re-authenticating...")
		loginToServiceAndSetContext()
		CHECK_COUNTER = 0
	}

	color.Cyan(fmt.Sprintf("\n > [ %s ] Getting status of litter robots...\n", getTimeString()))

	if err := API.FetchRobots(CTX); err != nil {
		color.Yellow(fmt.Sprintf("⚠️ [ %s ] Could not get robot details. Retrying in %d seconds...", getTimeString(), CHECK_INTERVAL))
		color.Yellow(" > " + fmt.Sprintf("%s", err.Error()))
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	} else {
		registeredRobots := API.Robots()
		for idx, r := range registeredRobots {
			color.Magenta(fmt.Sprintf("\n>>>>>>>>>>>>>>>>>>>>\n>> Robot %d of %d\n>>>>>>>>>>>>>>>>>>>>\n\n", idx+1, len(registeredRobots)))
			color.Magenta("\tRobot ID:")
			color.White(fmt.Sprintf("\t%s\n\n", r.LitterRobotID))

			color.Magenta("\tRobot Name:")
			color.White(fmt.Sprintf("\t%s\n\n", r.Name))

			unitStatus := r.UnitStatus
			statusText := mapUnitStatusToString(unitStatus)
			color.Magenta("\tRobot Status:\n")
			color.White(fmt.Sprintf("\t%s\n\n", statusText))

			if shouldSignalError(unitStatus) {
				color.Red("\t‼️  Robot needs attention\n\n")

				setLEDColor(ledColor.RGBA{255, 0, 0, 255}) // Red

			} else {
				color.Green("\t✅ Robot is happy :)\n\n")

				setLEDColor(ledColor.RGBA{0, 255, 0, 255}) // Green

			}
		}
	}

	color.Cyan(fmt.Sprintf(" > [ %s ] Waiting %d seconds before checking again...\n\n", getTimeString(), CHECK_INTERVAL))

	CHECK_COUNTER++

}

// Utility functions
func shouldSignalError(status float64) bool {
	return (status >= 3 && status < 6) || (status == 7) || (status >= 10 && status <= 12)
}

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

func getTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func handleShutdown(c *ws281x.Canvas) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		color.Red("\nShutting down gracefully...")
		setLEDColor(ledColor.RGBA{0, 0, 0, 255}) // Black
		c.Render()
		time.Sleep(100 * time.Millisecond)
		c.Close()
		os.Exit(0)
	}()
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
	fatal(err)
	defer c.Close()

	err = c.Initialize()
	fatal(err)

	handleShutdown(c)
	setLEDColor(ledColor.RGBA{0, 0, 255, 255}) // Blue

	go animate(c)
	loginToServiceAndSetContext()

	for {
		checkStatusOfRobot()
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	}
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
