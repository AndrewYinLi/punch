package main

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
	"strconv"
	"strings"
	"time"
	"os"
)

// Convert 24-hour time to 12-hour time
func convertTime(international string) string{
	internationalSplit := strings.Split(international, ":")
	hour, _ := strconv.Atoi(internationalSplit[0])
	if hour > 12{
		return strconv.Itoa(hour - 12) + ":" + internationalSplit[1] + " PM"
	}else{
		return international + " AM"
	}
}

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

// Append multiple byte slices into one byte slice
func multiAppend(slices [][]byte) []byte{
	var temp []byte
	for _, slice := range slices { // foreach (I'm a Go noob)
		temp = append(append(temp, slice...), []byte(",")...)
	}
	return temp[:len(temp)-1] // Remove trailing ","
}

// Record time in or out
func punch(){
	// Open `hours.db` in cd. Creates the database if it doesn't exist.
	db, err := bolt.Open("hours.db", 0600, nil)
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
			fmt.Print("Punched in at " + convertTime(currentTime) + ".\n")
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
			fmt.Print("Punched in at " + inTime + ".\n")
			fmt.Print("Punched out at " + outTime +  ".\n")
			fmt.Print("Time worked today: " + timeWorked +  ".\n")
		}
		return nil
	})
	if err != nil {
		log.Fatal("Error: Could not log hours in database.")
	}
}

func reset(){
	db, err := bolt.Open("hours.db", 0600, nil) // Open db
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
		fmt.Println("In and out times have been reset for today!")
	} else if os.Args[1] == "help"{
		fmt.Println("Usage:")
		fmt.Println("Punch in with `punch` and out with `punch` again")
		fmt.Println("Set your in time with `punch in <hh:mm>`.")
		fmt.Println("Set your out time with `punch out <hh:mm>`.")
		fmt.Println("Reset your times for the day with `punch reset`.")
	}

}