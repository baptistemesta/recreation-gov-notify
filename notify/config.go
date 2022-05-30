package notify

import "time"

type Config struct {
	Debug          bool
	PollInterval   time.Duration
	SMSFrom        string
	EmailFrom      string
	SMSTo          string
	EmailTo        string
	Availabilities Availabilities
}
type Availabilities struct {
	Partial       bool
	From          string
	To            string
	CampgroundIDs string
}

////Park       string
