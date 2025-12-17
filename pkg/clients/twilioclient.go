package clients

import (
	"log"

	twilio "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// MessageCreator is an interface for creating Twilio messages.
// This allows for dependency injection and mocking in tests.
type MessageCreator interface {
	CreateMessage(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error)
}

type TwilioClient struct {
	api MessageCreator
}

// NewTwilioClient creates a new TwilioClient with the default Twilio REST client.
func NewTwilioClient() *TwilioClient {
	client := twilio.NewRestClient()
	return &TwilioClient{
		api: client.Api,
	}
}

// NewTwilioClientWithAPI creates a new TwilioClient with a custom MessageCreator.
// This is useful for testing with a mock implementation.
func NewTwilioClientWithAPI(api MessageCreator) *TwilioClient {
	return &TwilioClient{
		api: api,
	}
}

func (t TwilioClient) SendSms(from string, to string, body string, dryRun bool) (*twilioApi.ApiV2010Message, error) {
	if dryRun {
		log.Printf("DRY RUN: Would have sent SMS from %s to %s with body: %s", from, to, body)
		return nil, nil
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetBody(body)

	return t.api.CreateMessage(params)
}
