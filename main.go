package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
	bolt "go.etcd.io/bbolt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"path"
)

// getDatabasePath joins the path for `hours.db` according to OS
// the path is the same as where the src code for punch.go is stored
func getDatabasePath() string{
	databasePath := os.Getenv("GOPATH") // Init to GOPATH
	dirSlice := []string{"src", "github.com", "AndrewYinLi", "punch"} // Path to `hours.db` file
	for _,dir := range dirSlice{
		databasePath = path.Join(databasePath, dir)
	}
	databasePath = path.Join(databasePath, "hours.db")
	return databasePath
}

// Print is a wrapper for nested function calls to print string input
// with the correct color designated by string fg
func Print(input string, fg string){
	switch fg { // Get foreground color
		case "red":
			color.Set(color.FgRed)
		case "green":
			color.Set(color.FgGreen)
		case "yellow":
			color.Set(color.FgYellow)
		case "blue":
			color.Set(color.FgBlue)
		case "magenta":
			color.Set(color.FgMagenta)
		case "cyan":
			color.Set(color.FgCyan)
	}
	fmt.Println(emoji.Sprint(input))
	color.Set(color.FgWhite) // Reset to white
}

// convertTime takes string international in 24-hour time (hh:mm) and
// returns the corresponding 12-hour time (hh:mm AM/PM) as a string
func convertTime(international string) string{
	internationalSplit := strings.Split(international, ":")
	hour, _ := strconv.Atoi(internationalSplit[0])
	if hour > 12{
		return strconv.Itoa(hour - 12) + ":" + internationalSplit[1] + " PM"
	}else{
		return international + " AM"
	}
}

// calcTimeWorked takes strings in and out and returns time
// worked in the format "X hours & Y minutes"
func calcTimeWorked(in string, out string) string{
	// Edge case example: in and out are the same
	if in == out {
		return "0 hours & 0 minutes"
	}
	inSplit := strings.Split(in, ":")
	outSplit := strings.Split(out, ":")
	inHour, _ := strconv.Atoi(inSplit[0])
	outHour, _ := strconv.Atoi(outSplit[0])
	hoursWorked := outHour - inHour
	// There's also " AM" or " PM" following the minutes so we use the slice operator
	inMinute, _ := strconv.Atoi(inSplit[1][0:2])
	outMinute, _ := strconv.Atoi(outSplit[1][0:2])
	var minutesWorked int
	// Edge case example = 6:27pm - 5:45 = 0:42
	if outMinute > inMinute {
		minutesWorked = outMinute - inMinute
	} else {
		hoursWorked--
		minutesWorked = 60 - inMinute + outMinute
	}
	return strconv.Itoa(hoursWorked) + " hours & " + strconv.Itoa(minutesWorked) + " minutes"
}

// multiAppend takes a slice of byte slices and
// concatenates the byte slices into one byte slice
func multiAppend(slices [][]byte) []byte{
	var temp []byte
	for _, slice := range slices { // foreach (I'm a Go noob)
		temp = append(append(temp, slice...), []byte(",")...)
	}
	return temp[:len(temp)-1] // Remove trailing ","
}

// punch records today's time in or time out if time in has already been recorded
func punch(){
	// Open `hours.db` designated path. Creates the database if it doesn't exist.
	db, err := bolt.Open(getDatabasePath(), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		// Create bucket to hold today's in and out time if it doesn't exist
		logBucket, err := tx.CreateBucketIfNotExists([]byte("log"))
		if err != nil {
			log.Fatal(err)
		}
		timeDate := time.Now().Format("2006-01-02 15:04")
		timeDateSplit := strings.Split(timeDate, " ")
		currentTime := timeDateSplit[1]
		currentDate :=  timeDateSplit[0] // Use today's date as key
		hoursWorked := logBucket.Get([]byte(currentDate)) // Value from bucket is hours worked
		if hoursWorked == nil || len(hoursWorked) == 0 { // Punch in because no time has been logged for today
			err := logBucket.Put([]byte(currentDate), []byte(currentTime))
			if err != nil {
				log.Fatal(err)
			}
			Print("Punched:punch: in at " + convertTime(currentTime) + ".", "green")
		} else{ // Punch out because time in was previously logged
			hoursWorked := logBucket.Get([]byte(currentDate))
			hoursWorkedSplit := strings.Split(string(hoursWorked), ",")
			inTime := convertTime(hoursWorkedSplit[0])
			var outTime string
			var timeWorked string
			if len(hoursWorkedSplit) == 1 { // If user hasn't punched out yet
				outTime = convertTime(currentTime)
				timeWorked = calcTimeWorked(inTime, outTime)
				err := logBucket.Put([]byte(currentDate), multiAppend([][]byte{hoursWorked, []byte(currentTime), []byte(timeWorked)}))
				if err != nil {
					log.Fatal(err)
				}
			} else{ // If user has already punched out
				outTime = convertTime(hoursWorkedSplit[1])
				timeWorked = hoursWorkedSplit[2]
			}
			Print("Punched:punch: in at " + inTime + ".", "green")
			Print("Punched:punch: out at " + outTime +  ".", "red")
			Print("Time worked today:hourglass:: " + timeWorked +  ".", "magenta")
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error: Could not log hours in database.")
	}
}

// export writes all logged dates, times in and out, and hours worked to "hours.csv"
func export(){
	db, err := bolt.Open(getDatabasePath(), 0600, nil) // Open db
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		logBucket := tx.Bucket([]byte("log")) // Get bucket with logged times
		if logBucket == nil{
			return nil
		}
		// Create output file
		fo, err := os.Create("hours.csv")
		if err != nil {
			return err
		}
		defer fo.Close()
		// Iterate over all logged times and write to file
		if err := logBucket.ForEach(func(k, v []byte) error {
			fmt.Fprintf(fo, string(multiAppend([][]byte{k, v})) + "\n")
			return nil
		});
		err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error: Could not export hours.")
	}
	Print("Exported 'hours.csv' to the current directory.", "white")
}

// reset deletes the times recorded for in and out only for today
func reset(){
	db, err := bolt.Open(getDatabasePath(), 0600, nil) // Open db
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		logBucket := tx.Bucket([]byte("log")) // Get bucket with logged times
		if logBucket == nil{
			return nil
		}
		// Get date and reformat
		timeDate := time.Now().Format("2006-01-02 15:04")
		currentDate := []byte(strings.Split(timeDate, " ")[0])
		// Use reformatted date as key in bucket
		hoursWorked := logBucket.Get(currentDate)
		if hoursWorked == nil{
			return nil
		}
		err := logBucket.Put(currentDate, nil) // Clear times
		if err != nil {
			log.Fatal(err)
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error: Could not log hours in database.")
	}
}

func main() {
	// Should probably reformat with an arg parser library
	if len(os.Args) == 1{
		punch()
	} else if os.Args[1] == "in" || os.Args[1] == "out"{
		if len(os.Args) != 2{
			log.Fatal("Error: Incorrect number of arguments. Usage: `punch " + os.Args[1] + " <hh:mm>`.")
		}
	} else if os.Args[1] == "reset"{
		reset()
		Print("Punch:punch: in and punch:punch: out times have been reset:recycle: for today!", "cyan")
	} else if os.Args[1] == "export"{
		export()
	} else if os.Args[1] == "help"{
		Print("","")
		Print("Usage:information_source::","blue")
		Print("Punch:punch: in with `punch` and punch:punch: out by calling `punch` again.", "yellow")
		Print("Also call `punch` after punching:punch: out for the day to see hours worked:chart_with_upwards_trend:.", "yellow")
		Print("Export your hours to a .csv file with `punch export`.", "magenta")
		Print("Set your in time with `punch in <hh:mm>`.", "green")
		Print("Set your out time with `punch out <hh:mm>`.", "red")
		Print("Reset:recycle: your times for the day with `punch reset`.", "cyan")
	}

}