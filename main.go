package main

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
	"strings"
	"time"
)

// Record time in
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
		if(hoursWorked == nil) { // Punch in because no time has been logged for today
			err := logBucket.Put([]byte(currentDate), []byte(currentTime))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Punched in at", currentTime)
		} else{ // Punch out because time in was previously logged
			err := logBucket.Put([]byte(currentDate), append(logBucket.Get([]byte(currentDate)), append([]byte(" "), []byte(currentTime)...)...))
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Punched out at", currentTime)
		}

		return nil
	})
	if err != nil {
		log.Fatal("Error: Could not log hours in database.")
	}
}

func main() {
	punch()

}