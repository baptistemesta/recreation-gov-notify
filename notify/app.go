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
func (a *App) Poll(ctx context.Context, campgrounds []Campground, start, end time.Time) (availabilities []string, err error) {
	t := time.NewTicker(a.cfg.PollInterval)

	for {
		select {
		case <-t.C:
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
						return nil, fmt.Errorf("Couldn't retrieve availabilities: %w", err)
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
				var results []string
			Outer:
				for site, dates := range available {
					start = initial
					for !start.After(end) {
						date := fmt.Sprintf("%sT00:00:00Z", start.Format("2006-01-02"))
						a.log.Debug(fmt.Sprintf("Cheking if %s is available for %s", site, date))
						if dates[date] {
							a.log.Debug(fmt.Sprintf("%s is available for %s", site, date))
							start = start.AddDate(0, 0, 1)
						} else {
							a.log.Debug(fmt.Sprintf("%s is NOT available for %s", site, date))
							continue Outer
						}
					}

					a.log.Info(fmt.Sprintf("%s is available!", site))
					results = append(results, site)
				}

				if len(results) > 0 {
					return results, nil
				}
			}
			a.log.Info("Sorry, no available campsites were found for your dates. We'll try again.")

		case <-ctx.Done():
			return nil, nil
		}
	}
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
