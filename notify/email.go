/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package notify

import (
	"fmt"
	"os"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type EmailNotifier struct {
	log       log15.Logger
	fromEmail string
}

func NewEmailNotifier(log log15.Logger, fromEmail string) *EmailNotifier {
	return &EmailNotifier{
		log:       log,
		fromEmail: fromEmail,
	}
}

func (n *EmailNotifier) Notify(to string, newAvailabilities []Availability) error {
	from := mail.NewEmail("Recreation.gov Notifier", n.fromEmail)
	subject := "Good news! Your campground is available"
	toAddr := mail.NewEmail(to, to)

	var sites string
	for _, newAvailability := range newAvailabilities {
		sites += fmt.Sprintf("- %s (%v): Site %s available %s on dates %s\n",
			newAvailability.campground,
			newAvailability.campgroundId,
			newAvailability.site,
			strings.Join(newAvailability.dates, ", "))
	}
	content := fmt.Sprintf(`
	New sites available!

	Sites:
	%s`, sites)
	plainTextContent := content
	htmlContent := content
	message := mail.NewSingleEmail(from, subject, toAddr, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	resp, err := client.Send(message)
	if err != nil {
		return err
	}

	n.log.Debug("Email sent", "status", resp.StatusCode, "to", to)
	return nil
}
