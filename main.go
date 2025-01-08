package main

import (
	"fmt"
	"os"
	"time"
	"context"
	"strconv"

	"github.com/joho/godotenv"
	robot "github.com/tlkamp/litter-api/v2/pkg/client"
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
		fmt.Println("Error logging in - ", err)
		os.Exit(1)
	}

	// Fetch the robots
	if err := api.FetchRobots(ctx); err != nil {
		fmt.Println("Error fetching robots - ", err)
		os.Exit(1)
	}

	// Loop through each robot and display details
	for _, r := range api.Robots() {
		fmt.Printf("Robot ID: %s\n", r.LitterRobotID)
		fmt.Printf("Name: %s\n", r.Name)

		// Fetch unit status from the robot struct
		unitStatus := r.UnitStatus

		// Map unit status to human-readable string
		statusText := mapUnitStatusToString(unitStatus)
		fmt.Printf("Unit Status: %s\n", statusText)

		shouldSignalError := shouldSignalError(unitStatus)

		fmt.Println("Needs attention?", shouldSignalError)

		fmt.Printf("Waiting %d seconds before checking again...", CHECK_INTERVAL)
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))

		checkStatusOfRobot()

		/*// Optionally, initiate a cycle for each robot
		err := api.Cycle(ctx, r.LitterRobotID)
		if err != nil {
			fmt.Printf("Error initiating cycle for %s: %v\n", r.Name, err)
		} else {
			fmt.Printf("Cycle initiated for %s\n", r.Name)
		}

		fmt.Println("---------------")*/
	}

}

func main() {

	// Load environment variables from .env file
	dotenvErr := godotenv.Load()
	if dotenvErr != nil {
		fmt.Println("No .env file detected. Defaulting to system env-vars instead.")
	}

	// Retrieve credentials from environment variables
	USERNAME = os.Getenv("ROBOT_EMAIL")
	PASSWORD = os.Getenv("ROBOT_PASS")
	
	if USERNAME == "" {
		fmt.Println("No username detected by script")
		os.Exit(1)
	}
	
	if PASSWORD == "" {
		fmt.Println("No password detected by script")
		os.Exit(1)
	}
	
	interval := os.Getenv("CHECK_INTERVAL")

	if interval == "" {
		fmt.Println("No interval set.")
		CHECK_INTERVAL = 5
	} else {
		value, convErr := strconv.Atoi(interval)

		if convErr != nil {
			CHECK_INTERVAL = 5
		} else {
			CHECK_INTERVAL = value
		}

	}

	checkStatusOfRobot()

}