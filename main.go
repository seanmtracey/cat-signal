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
var CHECK_COUNTER int

var LOGIN_RETRY_DELAY float64 = 5.0
var API *robot.Client
var CTX context.Context

var LED_DIRECTION = []int{ 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, }
var LED_VALUES    = []int{ 21, 42, 63, 84, 105, 126, 147, 168, 189, 210, 231, 252, }

func animate() {

	fmt.Print("[ ")

	for idx, _ := range LED_VALUES{
		
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
			
		fmt.Printf("%d, ", LED_VALUES[idx])
	}

	fmt.Print(" ]\n")

}

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

	if status >= 3 && status < 6 || status == 7 || status >= 11 && status <= 12 {
		return true
	}

	return false
}

// mapUnitStatusToString converts unit status codes to human-readable strings
func mapUnitStatusToString(status float64) string {
	
	statusMap := map[float64]string{
		0:  "(RDY) Ready.",
		1:  "(CCP) Clean Cycle in Progress.",
		2:  "(CCC) Clean Cycle Complete.",
		3:  "(CSF) Cat Sensor Fault.",
		4:  "(DF1) Drawer full; Will still cycle.",
		5:  "(DF2) Drawer full; Will still cycle",
		6:  "(CST) Cat Sensor Timing.",
		7:  "(CSI) Cat Sensor Interrupt.",
		8:  "(BR) Bonnet Removed.",
		9:  "(P) Paused.",
		10: "(OFF) Off.",
		11: "(SDF) Drawer full; Will not cycle.",
		12: "(DFS) Drawer full; Will not cycle.",
	}

	if description, ok := statusMap[status]; ok {
		return description
	}

	return "Unknown Status"

}

func checkStatusOfRobot(){

	totalTimeSinceLastLogin := time.Second * time.Duration(CHECK_COUNTER * CHECK_INTERVAL) 

	if totalTimeSinceLastLogin >= time.Minute * 10 {
		color.Yellow(" > Session is too old. Re-authenticating...")
		loginToServiceAndSetContext()
		CHECK_COUNTER = 0
	}

	color.Cyan(fmt.Sprintf("\n > [ %s ] Getting status of litter robots...\n", getTimeString()))

	// Fetch the robots
	if err := API.FetchRobots(CTX); err != nil {

		color.Yellow(fmt.Sprintf("⚠️ [ %s ] Could not get robot details. Retrying in %d seconds...", getTimeString(), CHECK_INTERVAL))
		color.Yellow(" > " + fmt.Sprintf("%s", err.Error()))
		time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
		
	} else {

		registeredRobots := API.Robots()

		// Loop through each robot and display details
		for idx, r := range registeredRobots {
			color.Magenta(fmt.Sprintf("\n>>>>>>>>>>>>>>>>>>>>\n>> Robot %d of %d\n>>>>>>>>>>>>>>>>>>>>\n\n", idx + 1, len(registeredRobots) ))
			color.Magenta("\tRobot ID:")
			color.White(fmt.Sprintf("\t%s\n\n", r.LitterRobotID))
			
			color.Magenta("\tRobot Name:")
			color.White(fmt.Sprintf("\t%s\n\n", r.Name))
	
			// Fetch unit status from the robot struct
			unitStatus := r.UnitStatus
	
			// Map unit status to human-readable string
			statusText := mapUnitStatusToString(unitStatus)
			color.Magenta("\tRobot Status:\n")
			color.White(fmt.Sprintf("\t%s\n\n", statusText))
	
			shouldSignalError := shouldSignalError(unitStatus)
	
			if shouldSignalError {
				color.Magenta("\tAction needed?")
				color.Red("\t‼️  Robot needs attention\n\n\n")
			} else {
				color.Green("\n\t✅ Robot is happy :)\n\n\n")
			}
	
			color.Cyan(fmt.Sprintf(" > [ %s ] Waiting %d seconds before checking again...\n\n", getTimeString(), CHECK_INTERVAL))
			time.Sleep(time.Second * time.Duration(CHECK_INTERVAL))
	
		}

	}
	
	CHECK_COUNTER += 1

}

func main() {

	// Load environment variables from .env file
	dotenvErr := godotenv.Load()
	if dotenvErr != nil {
		color.Cyan(">  No .env file detected. Defaulting to system env-vars instead.")
	}

	// Animate LEDs when they're wired up.
	/*for {
		animate()
		time.Sleep((1000 / 40) * time.Millisecond)
	}*/

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

	for{
		checkStatusOfRobot()
	}


}