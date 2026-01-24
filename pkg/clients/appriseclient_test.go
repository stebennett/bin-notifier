package clients

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockHTTPClient is a mock implementation of HTTPClient for testing
type mockHTTPClient struct {
	doFunc   func(req *http.Request) (*http.Response, error)
	doCalled bool
	lastReq  *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.doCalled = true
	m.lastReq = req
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}, nil
}

func TestSendNotification_DryRun_ReturnsNilWithoutCallingAPI(t *testing.T) {
	mock := &mockHTTPClient{}
	client := NewAppriseClientWithHTTP(mock)

	err := client.SendNotification("http://apprise:8000/notify/", "Test message", "", true)

	assert.Nil(t, err)
	assert.False(t, mock.doCalled, "HTTP client should not be called in dry-run mode")
}

func TestSendNotification_DryRunWithTag_ReturnsNilWithoutCallingAPI(t *testing.T) {
	mock := &mockHTTPClient{}
	client := NewAppriseClientWithHTTP(mock)

	err := client.SendNotification("http://apprise:8000/notify/", "Test message", "sms", true)

	assert.Nil(t, err)
	assert.False(t, mock.doCalled, "HTTP client should not be called in dry-run mode")
}

func TestSendNotification_CallsAppriseWithCorrectParams(t *testing.T) {
	mock := &mockHTTPClient{}
	client := NewAppriseClientWithHTTP(mock)

	appriseURL := "http://apprise:8000/notify/"
	body := "Test message body"

	err := client.SendNotification(appriseURL, body, "", false)

	assert.Nil(t, err)
	assert.True(t, mock.doCalled, "HTTP client should be called")
	assert.NotNil(t, mock.lastReq)
	assert.Equal(t, "POST", mock.lastReq.Method)
	assert.Equal(t, appriseURL, mock.lastReq.URL.String())
	assert.Equal(t, "application/json", mock.lastReq.Header.Get("Content-Type"))

	// Verify request body contains expected fields
	reqBody, _ := io.ReadAll(mock.lastReq.Body)
	assert.Contains(t, string(reqBody), `"urls"`)
	assert.Contains(t, string(reqBody), `"body"`)
	assert.Contains(t, string(reqBody), body)
	// Tag should be omitted when empty
	assert.NotContains(t, string(reqBody), `"tag"`)
}

func TestSendNotification_IncludesTagWhenProvided(t *testing.T) {
	mock := &mockHTTPClient{}
	client := NewAppriseClientWithHTTP(mock)

	appriseURL := "http://apprise:8000/notify/"
	body := "Test message body"
	tag := "sms"

	err := client.SendNotification(appriseURL, body, tag, false)

	assert.Nil(t, err)
	assert.True(t, mock.doCalled, "HTTP client should be called")

	// Verify request body contains tag field
	reqBody, _ := io.ReadAll(mock.lastReq.Body)
	assert.Contains(t, string(reqBody), `"tag":"sms"`)
}

func TestSendNotification_ReturnsErrorOnHTTPFailure(t *testing.T) {
	expectedError := errors.New("network error")
	mock := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, expectedError
		},
	}
	client := NewAppriseClientWithHTTP(mock)

	err := client.SendNotification("http://apprise:8000/notify/", "Test message", "", false)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to send notification")
}

func TestSendNotification_ReturnsErrorOnNon2xxStatus(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewBufferString("Bad request")),
			}, nil
		},
	}
	client := NewAppriseClientWithHTTP(mock)

	err := client.SendNotification("http://apprise:8000/notify/", "Test message", "", false)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "status 400")
	assert.Contains(t, err.Error(), "Bad request")
}

func TestSendNotification_SucceedsOn2xxStatus(t *testing.T) {
	statuses := []int{200, 201, 204}

	for _, status := range statuses {
		t.Run(string(rune(status)), func(t *testing.T) {
			mock := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: status,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				},
			}
			client := NewAppriseClientWithHTTP(mock)

			err := client.SendNotification("http://apprise:8000/notify/", "Test message", "", false)

			assert.Nil(t, err)
		})
	}
}

func TestNewAppriseClient_CreatesClientWithDefaultHTTP(t *testing.T) {
	client := NewAppriseClient()

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
}
