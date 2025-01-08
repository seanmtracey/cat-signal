package main

import (
	"fmt"
	"os"
	"time"
	"context"
	"strconv"
	"math"

	"github.com/joho/godotenv"
	robot "github.com/tlkamp/litter-api/v2/pkg/client"
	"github.com/fatih/color"
)

var USERNAME string
var PASSWORD string
var CHECK_INTERVAL int

var LOGIN_RETRY_DELAY float64 = 5.0
var API *robot.Client
var CTX context.Context

var LED_MAP = map[string][]int{
	"RED"    : []int{255, 0, 0},
	"GREEN"  : []int{0, 255, 0},
	"BLUE"   : []int{0, 0, 255},
	"YELLOW" : []int{255, 255, 0},
}

func getTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func loginToServiceAndSetContext(){
	
	API = robot.New(USERNAME, PASSWORD)
	CTX = context.Background()

	color.Cyan(" > Logging in to service and setting context for requests.")

	// Log in to the API
	err := API.Login(CTX)
	if err != nil {
		
		LOGIN_RETRY_DELAY = math.Round(LOGIN_RETRY_DELAY * 1.25)

		if  LOGIN_RETRY_DELAY > 300{
			LOGIN_RETRY_DELAY = 300
		}

		color.Red("🚫 Could not login to robot.")		
		color.Red(" > " + fmt.Sprintf("%s", err.Error()))
		color.Red(fmt.Sprintf(" > [ %s ] Waiting %f seconds before retrying login...\n\n", getTimeString(), LOGIN_RETRY_DELAY))

		time.Sleep(time.Second * time.Duration(LOGIN_RETRY_DELAY))
		loginToServiceAndSetContext()

	} else {

		color.Green("✅ Logged-in to robot service.")
		LOGIN_RETRY_DELAY = 5.0

	}

}

func shouldSignalError(status float64) bool {

	if status >= 3 && status < 6 || status >= 11 && status <= 12 {
		return true
	}

	return false
}

// mapUnitStatusToString converts unit status codes to human-readable strings
func mapUnitStatusToString(status float64) string {
	statusMap := map[float64]string{
		0:  "Ready.",
		1:  "Clean Cycle in Progress.",
		2:  "Clean Cycle Complete.",
		3:  "Cat Sensor Fault.",
		4:  "Drawer full; Will still cycle.",
		5:  "Drawer full; Will still cycle",
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

func checkStatusOfRobot(checkCounter int){

	totalTimeSinceLastLogin := time.Second * time.Duration(checkCounter * CHECK_INTERVAL) 

	if totalTimeSinceLastLogin >= time.Minute * 10 {
		color.Cyan(" > Session is too old. Re-authenticating...")
		loginToServiceAndSetContext()
		checkCounter = 0
	}

	// Fetch the robots
	if err := API.FetchRobots(CTX); err != nil {
		color.Yellow(fmt.Sprintf("⚠️ [ %s ] Could not get robot details. Retrying in %d seconds...", getTimeString(), CHECK_INTERVAL))
		color.Yellow(" > " + fmt.Sprintf("%s", err.Error()))
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
		checkStatusOfRobot(checkCounter + 1)
		
	} else {

		// Loop through each robot and display details
		for idx, r := range API.Robots() {
			color.Magenta(fmt.Sprintf("\n>>>>>>>>>>\n>> Robot %d:\n>>>>>>>>>>\n\n", idx))
			color.Magenta("\tRobot ID:")
			color.White(fmt.Sprintf("\t%s\n\n", r.LitterRobotID))
			
			color.Magenta("\tName:")
			color.White(fmt.Sprintf("\t%s\n\n", r.Name))
	
			// Fetch unit status from the robot struct
			unitStatus := r.UnitStatus
	
			// Map unit status to human-readable string
			statusText := mapUnitStatusToString(unitStatus)
			color.Magenta("\tUnit Status:\n")
			color.White(fmt.Sprintf("\t%s\n\n", statusText))
	
			shouldSignalError := shouldSignalError(unitStatus)
	
			if shouldSignalError {
				color.Magenta("\tAction needed?")
				color.Red("\t‼️  Robot needs attention\n\n")
			} else {
				color.Green("\n\t✅ Robot is happy :)\n\n")
			}
	
			color.Cyan(fmt.Sprintf(" > [ %s ] Waiting %d seconds before checking again...\n\n", getTimeString(), CHECK_INTERVAL))
			time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	
		}

		checkStatusOfRobot(checkCounter + 1)

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

		color.Green(fmt.Sprintf("✅ Checking interval set to %d seconds.", CHECK_INTERVAL))

	}

	loginToServiceAndSetContext()
	checkStatusOfRobot(0)

}