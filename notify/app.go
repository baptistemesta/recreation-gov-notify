/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package notify

import (
	"context"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
)

type App struct {
	cfg *Config
	log log15.Logger

	client        *Client
	smsNotifier   Notifier
	emailNotifier Notifier

	availabilities []Availability
}

type Availability struct {
	campground   string
	campgroundId string
	site         string
	dates        []string
}

type Notifier interface {
	Notify(to string, campgroundName, checkInDate, checkOutDate string, available []string) error
}

func New(log log15.Logger, cfg *Config) *App {
	return &App{
		cfg:           cfg,
		log:           log,
		client:        NewClient(log),
		smsNotifier:   NewSMSNotifier(log, cfg.SMSFrom),
		emailNotifier: NewEmailNotifier(log, cfg.SMSFrom),
	}
}

func (a *App) Search(query string) ([]Campground, error) {
	return a.client.Search(query)
}

// Poll is a blocking operation. To poll multiple campgrounds call this method
// in its own goroutine.
func (a *App) Poll(ctx context.Context, campgrounds []Campground, start, end time.Time) {
	t := time.NewTicker(a.cfg.PollInterval)

	for {
		select {
		case <-t.C:

			err := a.executeSearch(campgrounds, start, end)
			if err != nil {
				a.log.Error("There was an unrecoverable error, will retry on the next tick", "err", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (a *App) executeSearch(campgrounds []Campground, start time.Time, end time.Time) error {
	var newAvailabilities []Availability
	for _, campground := range campgrounds {
		a.log.Debug(fmt.Sprintf("Cheking for avaibility in campground %s", campground.Name))
		curPeriod := fmt.Sprintf("%d-%02d", start.Year(), start.Month())
		endPeriod := fmt.Sprintf("%d-%02d", end.Year(), end.Month())

		var months []string
		months = append(months, curPeriod)

		// Determine months in date range.
		initial := start
		for curPeriod != endPeriod {
			start = start.AddDate(0, 1, 0)
			curPeriod = fmt.Sprintf("%d-%02d", start.Year(), start.Month())
			months = append(months, curPeriod)
		}
		start = initial

		// Build availability map.
		available := make(map[string]map[string]bool)
		for _, m := range months {
			campsites, err := a.client.Availability(campground.EntityID, m)
			if err != nil {
				return fmt.Errorf("Couldn't retrieve availabilities: %w", err)
			}

			a.log.Debug(fmt.Sprintf("Found %v day/campsites ", len(campsites)))
			for _, c := range campsites {
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

		// Check for contiguous availability.
		for site, dates := range available {
			start = initial
			var availableDates []string
			var numberOfDays = 0
			for !start.After(end) {
				numberOfDays++
				date := fmt.Sprintf("%sT00:00:00Z", start.Format("2006-01-02"))
				a.log.Debug(fmt.Sprintf("Cheking if %s is available for %s", site, date))
				if dates[date] {
					a.log.Debug(fmt.Sprintf("%s is available for %s", site, date))
					availableDates = append(availableDates, date)
				}
				start = start.AddDate(0, 0, 1)
			}
			if len(availableDates) > 0 {
				if !a.cfg.Availabilities.Partial && len(availableDates) != numberOfDays {
					continue
				}
				a.log.Debug(fmt.Sprintf("%s is available!", site))

				newAvailability := Availability{
					campground:   campground.Name,
					campgroundId: campground.EntityID,
					site:         site,
					dates:        availableDates,
				}
				var contains = false
				for _, availability := range a.availabilities {
					if availability.campgroundId == newAvailability.campgroundId &&
						availability.site == newAvailability.site {
						contains = true
					}
				}
				if !contains {
					newAvailabilities = append(newAvailabilities, newAvailability)
				}
			}
		}

	}

	if len(newAvailabilities) > 0 {
		a.log.Info("Congrats, new available campsites were found for your dates!",
			"availabilities", newAvailabilities)
		a.availabilities = append(a.availabilities, newAvailabilities...)
		a.notify(newAvailabilities)
	} else {
		a.log.Info("Sorry, no new available campsites were found for your dates. We'll try again.")
	}
	return nil
}

// TODO: This pattern feels a bit odd, but want to leave the notifiers decoupled
// for testing and in case we want to poll/notify for multiple requests (ie: if
// we add a webapp frontend or something).
func (a *App) SMSNotify(toNumber string, campgroundName, checkInDate, checkOutDate string, available []string) error {
	return a.smsNotifier.Notify(toNumber, campgroundName, checkInDate, checkOutDate, available)
}

func (a *App) EmailNotify(toEmail string, campgroundName, checkInDate, checkOutDate string, available []string) error {
	return a.emailNotifier.Notify(toEmail, campgroundName, checkInDate, checkOutDate, available)
}

func (a *App) notify(newAvailabilities []Availability) {
	smsTo := a.cfg.SMSTo
	emailTo := a.cfg.EmailTo

	if smsTo != "" {
		a.log.Info("Sending SMS", "to", smsTo)
		//err := a.SMSNotify(smsTo, "TODO", newAvailabilities.ava, checkOutDate, availabilities)
		//if err != nil {
		//	a.log.Error("Could not send SMS message", "err", err)
		//}
	}
	if emailTo != "" {
		a.log.Info("Sending SMS", "to", smsTo)
		//err := a.EmailNotify(emailTo, "TODO", checkInDate, checkOutDate, availabilities)
		//if err != nil {
		//	a.log.Error("Could not send SMS message", "err", err)
		//}
	}
	a.log.Info("Have a good trip!")
}
