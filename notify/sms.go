/*
Copyright © 2022 Kyle Chadha @kylechadha
*/
package notify

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type SMSNotifier struct {
	log        log15.Logger
	client     *twilio.RestClient
	fromNumber string
}

func NewSMSNotifier(log log15.Logger, fromNumber string) *SMSNotifier {
	return &SMSNotifier{
		log:        log,
		fromNumber: fromNumber,
		client:     twilio.NewRestClient(),
	}
}

const SMSTemplate = `
Good news from the (very unofficial) Recreation.gov Notifier!
The following sites are available:
%s`

func (n *SMSNotifier) Notify(to string, newAvailabilities []Availability) error {
	params := &openapi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(n.fromNumber)

	var sites string
	for _, newAvailability := range newAvailabilities {
		sites += fmt.Sprintf("- %s (%v): Site %s available\n",
			newAvailability.campground,
			newAvailability.campgroundId,
			newAvailability.site)
	}
	body := fmt.Sprintf(SMSTemplate, sites)
	params.SetBody(body)
	n.log.Debug(fmt.Sprintf("Will send SMS:\n%s", body))

	resp, err := n.client.Api.CreateMessage(params)
	if err != nil {
		return err
	}

	n.log.Debug("SMS message sent", "status", *resp.Status, "to", to)
	return nil
}
