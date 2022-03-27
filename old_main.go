package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
)

// ** HERE
// then connect to text/email
// then convert to CLI app
// - Split out front end vs back end here
// - How do we programmatically determine campground ID? Is there another API we can hit to search?
// then create web frontend
// productionize w/ cloud functions?
// - Do this when you built the front end, either we have one backend app that spins up goroutines to poll
// Or spins up a cloud function

// flags
// - polling interval
// - twilio SID + authToken + from number
// - your number
// - your email
// - debug mode
// - campsite ID

// ^ would be good to get viper so you can do env vars / CLI flags / or config file

// flow
// - if no campsite ID, search
// -- select campground
// - if available on first try, don't send a notification, exit
// - if unavailable
// -- how do you want to get notified? sms, email, or both?
// -- can put this information in now if you haven't already

func oldMain() {
	l := log15.New()
	client := http.Client{}

	from := os.Getenv("TWILIO_FROM")
	smsNotify := NewSMSNotifier(l, from)

	fmt.Println("Whatcha searching for? (Enter a campground or area)")
	displayed := false

	var campgrounds []Campground
	for !displayed {
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		searchQuery := strings.Replace(input, "\n", "", -1) // convert CRLF to LF

		var err error
		campgrounds, err = Suggest(searchQuery)
		if err != nil {
			log.Fatal(err)
		}

		for i, campground := range campgrounds {
			if !displayed {
				fmt.Println("Select the number that best matches")
				displayed = true
			}
			campgrounds[i].Name = strings.Title(strings.ToLower(campground.Name))
			fmt.Printf("[%d] %s\n", i+1, campgrounds[i].Name)
		}

		if !displayed {
			fmt.Println("Sorry! We didn't find any campgrounds with that query. Try again?")
		}
	}

	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		log.Fatal(err)
	}
	choice, err := strconv.Atoi(string(char))
	if err != nil {
		log.Fatal(err)
	}

	campground := campgrounds[choice-1]
	campgroundID, err := strconv.Atoi(campground.EntityID)
	if err != nil {
		log.Fatal(err)
	}

	checkInDate := "2022-06-01"
	checkOutDate := "2022-06-04"
	phoneNumber := "+18582310672"

	l.Info("Searching recreation.gov...", "campground", campground.Name, "checkIn", checkInDate, "checkOut", checkOutDate)

	st, err := time.Parse("2006-01-02", checkInDate)
	if err != nil {
		l.Error("Invalid check in date", "err", err)
		exitCode = 1
		return
	}
	etRaw, err := time.Parse("2006-01-02", checkOutDate)
	if err != nil {
		l.Error("Invalid check out date", "err", err)
		exitCode = 1
		return
	}
	et := etRaw.AddDate(0, 0, -1) // checkOutDate does not need to be available

	if st.After(et) {
		l.Error("Start date is after end date")
		exitCode = 1
		return
	}

	curPeriod := fmt.Sprintf("%d-%02d", st.Year(), st.Month())
	endPeriod := fmt.Sprintf("%d-%02d", et.Year(), et.Month())

	var months []string
	months = append(months, curPeriod)

	initial := st
	for curPeriod != endPeriod {
		st = st.AddDate(0, 1, 0)
		curPeriod = fmt.Sprintf("%d-%02d", st.Year(), st.Month())
		months = append(months, curPeriod)
	}
	st = initial

	available := make(map[string]map[string]bool)
	u := fmt.Sprintf(availabilityURL, campgroundID)
	for _, m := range months {
		l.Debug("Requesting data", "month", m)

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			log.Fatal(err)
		}

		q := url.Values{}
		q.Add("start_date", m+"-01T00:00:00.000Z")
		req.URL.RawQuery = q.Encode()

		// Need to spoof the user agent or CloudFront blocks us.
		req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}
			log.Fatal(resp.StatusCode, ": ", string(bytes))
		}

		var ar AvailabilityResponse
		err = json.NewDecoder(resp.Body).Decode(&ar)
		if err != nil {
			log.Fatal(err)
		}

		for _, c := range ar.Campsites {
			for date, a := range c.Availabilities {
				if a == "Available" {
					if available[c.Site] == nil {
						available[c.Site] = make(map[string]bool)
					}

					available[c.Site][date] = true
				}
			}
		}
	}

	var results []string
Outer:
	for site, dates := range available {
		st = initial
		for !st.After(et) {
			date := fmt.Sprintf("%sT00:00:00Z", st.Format("2006-01-02"))
			if dates[date] {
				l.Debug(fmt.Sprintf("Site %s available for %s", site, st.Format("2006-01-02")))
				st = st.AddDate(0, 0, 1)
			} else {
				continue Outer
			}
		}

		l.Info(fmt.Sprintf("Site %s is available", site))
		results = append(results, site)
	}

	if len(results) == 0 {
		l.Info("Sorry, no available campsites were found for your dates")
	} else {
		smsNotify.Notify(phoneNumber, campground.Name, checkInDate, checkOutDate, results)
	}
}
