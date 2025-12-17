package clients

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// mockMessageCreator is a mock implementation of MessageCreator for testing
type mockMessageCreator struct {
	createMessageFunc   func(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error)
	createMessageCalled bool
	lastParams          *twilioApi.CreateMessageParams
}

func (m *mockMessageCreator) CreateMessage(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error) {
	m.createMessageCalled = true
	m.lastParams = params
	if m.createMessageFunc != nil {
		return m.createMessageFunc(params)
	}
	return &twilioApi.ApiV2010Message{}, nil
}

func TestSendSms_DryRun_ReturnsNilWithoutCallingAPI(t *testing.T) {
	mock := &mockMessageCreator{}
	client := NewTwilioClientWithAPI(mock)

	result, err := client.SendSms("+1234567890", "+0987654321", "Test message", true)

	assert.Nil(t, err)
	assert.Nil(t, result)
	assert.False(t, mock.createMessageCalled, "API should not be called in dry-run mode")
}

func TestSendSms_CallsTwilioWithCorrectParams(t *testing.T) {
	mock := &mockMessageCreator{}
	client := NewTwilioClientWithAPI(mock)

	from := "+1234567890"
	to := "+0987654321"
	body := "Test message body"

	_, err := client.SendSms(from, to, body, false)

	assert.Nil(t, err)
	assert.True(t, mock.createMessageCalled, "API should be called")
	assert.NotNil(t, mock.lastParams)
	assert.Equal(t, to, *mock.lastParams.To)
	assert.Equal(t, from, *mock.lastParams.From)
	assert.Equal(t, body, *mock.lastParams.Body)
}

func TestSendSms_ReturnsErrorFromTwilio(t *testing.T) {
	expectedError := errors.New("twilio API error")
	mock := &mockMessageCreator{
		createMessageFunc: func(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error) {
			return nil, expectedError
		},
	}
	client := NewTwilioClientWithAPI(mock)

	result, err := client.SendSms("+1234567890", "+0987654321", "Test message", false)

	assert.Nil(t, result)
	assert.Equal(t, expectedError, err)
}

func TestSendSms_ReturnsMessageFromTwilio(t *testing.T) {
	sid := "SM12345"
	expectedMessage := &twilioApi.ApiV2010Message{
		Sid: &sid,
	}
	mock := &mockMessageCreator{
		createMessageFunc: func(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error) {
			return expectedMessage, nil
		},
	}
	client := NewTwilioClientWithAPI(mock)

	result, err := client.SendSms("+1234567890", "+0987654321", "Test message", false)

	assert.Nil(t, err)
	assert.Equal(t, expectedMessage, result)
	assert.Equal(t, sid, *result.Sid)
}
