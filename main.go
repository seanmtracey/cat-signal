package main

import (
	"fmt"
	"os"
	"time"
	"context"
	"strconv"

	"github.com/joho/godotenv"
	robot "github.com/tlkamp/litter-api/v2/pkg/client"
	"github.com/fatih/color"
)

var USERNAME string
var PASSWORD string
var CHECK_INTERVAL int

func shouldSignalError(status float64) bool {

	if status >= 3 && status < 6 || status >= 11 && status <= 12 {
		return true
	}

	return false
}

// mapUnitStatusToString converts unit status codes to human-readable strings
func mapUnitStatusToString(status float64) string {
	statusMap := map[float64]string{
		0:  "Ready",
		1:  "Clean Cycle in Progress",
		2:  "Clean Cycle Complete",
		3:  "Cat Sensor Fault",
		4:  "Drawer full - will still cycle",
		5:  "Drawer full - will still cycle",
		6:  "Cat Sensor Timing",
		7:  "Cat Sensor Interrupt",
		8:  "Bonnet Removed",
		9:  "Paused",
		10: "Off",
		11: "Drawer full - will not cycle",
		12: "Drawer full - will not cycle",
	}
	if description, ok := statusMap[status]; ok {
		return description
	}
	return "Unknown Status"
}

func checkStatusOfRobot(){

	// Initialize API client
	api := robot.New(USERNAME, PASSWORD)
	ctx := context.Background()

	// Log in to the API
	err := api.Login(ctx)
	if err != nil {
		color.Red("🚫 Could not log-in to robot.")
		fmt.Println(err)
		os.Exit(1)
	} else {
		color.Green("✅ Logged-in to robot service.")
	}

	// Fetch the robots
	if err := api.FetchRobots(ctx); err != nil {
		color.Yellow(fmt.Sprintf("⚠️  Could not get robot details. Retrying in %d seconds...", CHECK_INTERVAL))
		fmt.Println(err)
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
		checkStatusOfRobot()
		return
	}

	// Loop through each robot and display details
	for _, r := range api.Robots() {
		color.Cyan(fmt.Sprintf("\tRobot ID: %s\n", r.LitterRobotID))
		color.Cyan(fmt.Sprintf("\tName: %s\n", r.Name))

		// Fetch unit status from the robot struct
		unitStatus := r.UnitStatus

		// Map unit status to human-readable string
		statusText := mapUnitStatusToString(unitStatus)
		color.Cyan(fmt.Sprintf("\tUnit Status: %s\n", statusText))

		shouldSignalError := shouldSignalError(unitStatus)

		if shouldSignalError {
			color.Yellow("⚠️  Robot needs attention")
		} else {
			color.Green("✅ Robot is happy :)")
		}

		color.Cyan(fmt.Sprintf("> Waiting %d seconds before checking again...", CHECK_INTERVAL))
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))

		checkStatusOfRobot()

	}

}

func main() {

	// Load environment variables from .env file
	dotenvErr := godotenv.Load()
	if dotenvErr != nil {
		color.Cyan(">  No .env file detected. Defaulting to system env-vars instead.")
	}

	// Retrieve credentials from environment variables
	USERNAME = os.Getenv("ROBOT_EMAIL")
	PASSWORD = os.Getenv("ROBOT_PASS")

	shouldExit := false
	
	if USERNAME == "" {
		color.Red("🚫 No username detected by script.")
		shouldExit = true
	} else {
		color.Green("✅ Username detected in environment.")
	}
	
	if PASSWORD == "" {
		color.Red("🚫 No password detected by script.")
		shouldExit = true
	} else {
		color.Green("✅ Password detected in environment.")
	}

	if shouldExit == true {
		os.Exit(1)
	}
	
	interval := os.Getenv("CHECK_INTERVAL")

	if interval == "" {
		color.Yellow("⚠️  No interval set. Defaulting to 60 seconds...")
		CHECK_INTERVAL = 60
	} else {
		value, convErr := strconv.Atoi(interval)

		if convErr != nil {
			CHECK_INTERVAL = 60
		} else {
			CHECK_INTERVAL = value
		}

		color.Green(fmt.Sprintf("✅ Checking intervale set to %d seconds.", CHECK_INTERVAL))

	}

	checkStatusOfRobot()

}