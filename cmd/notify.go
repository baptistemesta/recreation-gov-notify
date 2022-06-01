/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/kylechadha/recreation-gov-notify/notify"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
)

func runNotify(cfg *notify.Config) {
	ctx := context.Background()
	log := log15.New("module", "notify")
	if !cfg.Debug {
		log.SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.StdoutHandler))
	}
	log.Debug("Using config", "config", cfg)

	// ** Do we want to have a separate config for the CLI app that includes SMSTo and EmailTo, and then
	// embeds or includes the App config?

	// ** remove once set by CLI init
	cfg.PollInterval = time.Minute
	app := notify.New(log, cfg)
	reader := bufio.NewReader(os.Stdin)

	campgrounds := getCampGrounds(cfg, reader, app)
	checkInDate, start := getStart(cfg, reader)
	checkOutDate, end := getEnd(cfg, reader, start)

	fmt.Printf("Now we're in business! Searching recreation.gov availability for %x campgrounds from %s to %s\n", len(campgrounds), checkInDate, checkOutDate)
	app.Poll(ctx, campgrounds, start, end)

}

func getCampGrounds(cfg *notify.Config, reader *bufio.Reader, app *notify.App) []notify.Campground {
	var campgrounds []notify.Campground
	if cfg.Availabilities.CampgroundIDs == "" {
	Outer:
		for {
			fmt.Println("Which campground are you looking for?")

			reader.Reset(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Sorry, there was an error, please try again. Error : %w\n", err)
				continue
			}
			query := strings.Replace(input, "\n", "", -1) // Convert CRLF to LF.

			campgrounds, err := app.Search(query)
			if err != nil {
				fmt.Printf("Sorry, there was an error, please try again. Error : %w\n", err)
				continue
			}
			if len(campgrounds) == 0 {
				fmt.Println("Sorry, we didn't find any campgrounds for that query. Please try again")
				continue
			}

			fmt.Println("Select the number that best matches:")
			for i, c := range campgrounds {
				fmt.Printf("[%d] %s\n", i+1, c.Name)
			}
			lastIndex := len(campgrounds) + 1
			fmt.Printf("[%d] None of these, let me search again\n", lastIndex)

			for {
				reader.Reset(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Printf("Sorry, there was an error, please try again. Error : %w\n", err)
					continue
				}
				choice, err := strconv.Atoi(strings.Replace(input, "\n", "", -1))
				if err != nil || choice > lastIndex {
					fmt.Printf("Sorry, that was an invalid selection, please try again")
					continue
				}
				if choice == lastIndex {
					continue Outer
				}

				campgrounds = []notify.Campground{campgrounds[choice-1]}
				break Outer
			}
		}
	} else {
		campgroundIds := strings.Split(cfg.Availabilities.CampgroundIDs, ",")
		campgrounds = make([]notify.Campground, len(campgroundIds))
		for i, campgroundId := range campgroundIds {
			campground, err := app.Get(campgroundId)
			if err != nil {
				fmt.Printf("Sorry, there was an error, please try again.", "err", err)
			}
			campgrounds[i] = notify.Campground{
				EntityID: campground.EntityID,
				Name:     campground.Name,
			}
		}
	}
	fmt.Println("Will search for campgrounds")
	for _, campground := range campgrounds {
		fmt.Println(fmt.Sprintf("- %v (%v)", campground.Name, campground.EntityID))
	}
	return campgrounds
}

func getEnd(cfg *notify.Config, reader *bufio.Reader, start time.Time) (string, time.Time) {
	var checkOutDate = cfg.Availabilities.To
	var end time.Time

	if checkOutDate == "" {
		for {
			fmt.Println(`When's your check out? Please enter in "MM-DD-YYYY" format.`)

			reader.Reset(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Sorry, there was an error, please try again. Error : %w\n", err)
				continue
			}
			checkOutDate = strings.Replace(input, "\n", "", -1) // Convert CRLF to LF.

			endUnadjusted, err := time.Parse("01-02-2006", checkOutDate)
			if err != nil {
				fmt.Println("Sorry I couldn't parse that date. please try again. Error : %w\n", err)
				continue
			}
			end = endUnadjusted.AddDate(0, 0, -1) // checkOutDate does not need to be available.

			if start.After(end) {
				fmt.Println("Check out needs to be after check in ;)")
				continue
			}
			break
		}
	} else {
		end, _ = time.Parse("01-02-2006", checkOutDate)
	}
	return checkOutDate, end
}

func getStart(cfg *notify.Config, reader *bufio.Reader) (string, time.Time) {
	var checkInDate = cfg.Availabilities.From
	var start time.Time
	if checkInDate == "" {

		for {
			fmt.Println(`When's your check in? Please enter in "MM-DD-YYYY" format.`)

			reader.Reset(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Sorry, there was an error, please try again. Error : %w\n", err)
				continue
			}
			checkInDate = strings.Replace(input, "\n", "", -1) // Convert CRLF to LF.

			start, err = time.Parse("01-02-2006", checkInDate)
			if err != nil {
				fmt.Println("Sorry I couldn't parse that date. please try again. Error : %w\n", err)
				continue
			}
			break
		}
	} else {
		start, _ = time.Parse("01-02-2006", checkInDate)
	}
	return checkInDate, start
}
