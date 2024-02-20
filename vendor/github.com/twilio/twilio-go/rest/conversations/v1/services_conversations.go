/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Conversations
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/twilio/twilio-go/client"
)

// Optional parameters for the method 'CreateServiceConversation'
type CreateServiceConversationParams struct {
	// The X-Twilio-Webhook-Enabled HTTP request header
	XTwilioWebhookEnabled *string `json:"X-Twilio-Webhook-Enabled,omitempty"`
	// The human-readable name of this conversation, limited to 256 characters. Optional.
	FriendlyName *string `json:"FriendlyName,omitempty"`
	// An application-defined string that uniquely identifies the resource. It can be used to address the resource in place of the resource's `sid` in the URL.
	UniqueName *string `json:"UniqueName,omitempty"`
	// An optional string metadata field you can use to store any data you wish. The string value must contain structurally valid JSON if specified.  **Note** that if the attributes are not set \\\"{}\\\" will be returned.
	Attributes *string `json:"Attributes,omitempty"`
	// The unique ID of the [Messaging Service](https://www.twilio.com/docs/messaging/api/service-resource) this conversation belongs to.
	MessagingServiceSid *string `json:"MessagingServiceSid,omitempty"`
	// The date that this resource was created.
	DateCreated *time.Time `json:"DateCreated,omitempty"`
	// The date that this resource was last updated.
	DateUpdated *time.Time `json:"DateUpdated,omitempty"`
	//
	State *string `json:"State,omitempty"`
	// ISO8601 duration when conversation will be switched to `inactive` state. Minimum value for this timer is 1 minute.
	TimersInactive *string `json:"Timers.Inactive,omitempty"`
	// ISO8601 duration when conversation will be switched to `closed` state. Minimum value for this timer is 10 minutes.
	TimersClosed *string `json:"Timers.Closed,omitempty"`
	// The default email address that will be used when sending outbound emails in this conversation.
	BindingsEmailAddress *string `json:"Bindings.Email.Address,omitempty"`
	// The default name that will be used when sending outbound emails in this conversation.
	BindingsEmailName *string `json:"Bindings.Email.Name,omitempty"`
}

func (params *CreateServiceConversationParams) SetXTwilioWebhookEnabled(XTwilioWebhookEnabled string) *CreateServiceConversationParams {
	params.XTwilioWebhookEnabled = &XTwilioWebhookEnabled
	return params
}
func (params *CreateServiceConversationParams) SetFriendlyName(FriendlyName string) *CreateServiceConversationParams {
	params.FriendlyName = &FriendlyName
	return params
}
func (params *CreateServiceConversationParams) SetUniqueName(UniqueName string) *CreateServiceConversationParams {
	params.UniqueName = &UniqueName
	return params
}
func (params *CreateServiceConversationParams) SetAttributes(Attributes string) *CreateServiceConversationParams {
	params.Attributes = &Attributes
	return params
}
func (params *CreateServiceConversationParams) SetMessagingServiceSid(MessagingServiceSid string) *CreateServiceConversationParams {
	params.MessagingServiceSid = &MessagingServiceSid
	return params
}
func (params *CreateServiceConversationParams) SetDateCreated(DateCreated time.Time) *CreateServiceConversationParams {
	params.DateCreated = &DateCreated
	return params
}
func (params *CreateServiceConversationParams) SetDateUpdated(DateUpdated time.Time) *CreateServiceConversationParams {
	params.DateUpdated = &DateUpdated
	return params
}
func (params *CreateServiceConversationParams) SetState(State string) *CreateServiceConversationParams {
	params.State = &State
	return params
}
func (params *CreateServiceConversationParams) SetTimersInactive(TimersInactive string) *CreateServiceConversationParams {
	params.TimersInactive = &TimersInactive
	return params
}
func (params *CreateServiceConversationParams) SetTimersClosed(TimersClosed string) *CreateServiceConversationParams {
	params.TimersClosed = &TimersClosed
	return params
}
func (params *CreateServiceConversationParams) SetBindingsEmailAddress(BindingsEmailAddress string) *CreateServiceConversationParams {
	params.BindingsEmailAddress = &BindingsEmailAddress
	return params
}
func (params *CreateServiceConversationParams) SetBindingsEmailName(BindingsEmailName string) *CreateServiceConversationParams {
	params.BindingsEmailName = &BindingsEmailName
	return params
}

// Create a new conversation in your service
func (c *ApiService) CreateServiceConversation(ChatServiceSid string, params *CreateServiceConversationParams) (*ConversationsV1ServiceConversation, error) {
	path := "/v1/Services/{ChatServiceSid}/Conversations"
	path = strings.Replace(path, "{"+"ChatServiceSid"+"}", ChatServiceSid, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.FriendlyName != nil {
		data.Set("FriendlyName", *params.FriendlyName)
	}
	if params != nil && params.UniqueName != nil {
		data.Set("UniqueName", *params.UniqueName)
	}
	if params != nil && params.Attributes != nil {
		data.Set("Attributes", *params.Attributes)
	}
	if params != nil && params.MessagingServiceSid != nil {
		data.Set("MessagingServiceSid", *params.MessagingServiceSid)
	}
	if params != nil && params.DateCreated != nil {
		data.Set("DateCreated", fmt.Sprint((*params.DateCreated).Format(time.RFC3339)))
	}
	if params != nil && params.DateUpdated != nil {
		data.Set("DateUpdated", fmt.Sprint((*params.DateUpdated).Format(time.RFC3339)))
	}
	if params != nil && params.State != nil {
		data.Set("State", *params.State)
	}
	if params != nil && params.TimersInactive != nil {
		data.Set("Timers.Inactive", *params.TimersInactive)
	}
	if params != nil && params.TimersClosed != nil {
		data.Set("Timers.Closed", *params.TimersClosed)
	}
	if params != nil && params.BindingsEmailAddress != nil {
		data.Set("Bindings.Email.Address", *params.BindingsEmailAddress)
	}
	if params != nil && params.BindingsEmailName != nil {
		data.Set("Bindings.Email.Name", *params.BindingsEmailName)
	}

	if params != nil && params.XTwilioWebhookEnabled != nil {
		headers["X-Twilio-Webhook-Enabled"] = *params.XTwilioWebhookEnabled
	}

	resp, err := c.requestHandler.Post(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &ConversationsV1ServiceConversation{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}

// Optional parameters for the method 'DeleteServiceConversation'
type DeleteServiceConversationParams struct {
	// The X-Twilio-Webhook-Enabled HTTP request header
	XTwilioWebhookEnabled *string `json:"X-Twilio-Webhook-Enabled,omitempty"`
}

func (params *DeleteServiceConversationParams) SetXTwilioWebhookEnabled(XTwilioWebhookEnabled string) *DeleteServiceConversationParams {
	params.XTwilioWebhookEnabled = &XTwilioWebhookEnabled
	return params
}

// Remove a conversation from your service
func (c *ApiService) DeleteServiceConversation(ChatServiceSid string, Sid string, params *DeleteServiceConversationParams) error {
	path := "/v1/Services/{ChatServiceSid}/Conversations/{Sid}"
	path = strings.Replace(path, "{"+"ChatServiceSid"+"}", ChatServiceSid, -1)
	path = strings.Replace(path, "{"+"Sid"+"}", Sid, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.XTwilioWebhookEnabled != nil {
		headers["X-Twilio-Webhook-Enabled"] = *params.XTwilioWebhookEnabled
	}

	resp, err := c.requestHandler.Delete(c.baseURL+path, data, headers)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

// Fetch a conversation from your service
func (c *ApiService) FetchServiceConversation(ChatServiceSid string, Sid string) (*ConversationsV1ServiceConversation, error) {
	path := "/v1/Services/{ChatServiceSid}/Conversations/{Sid}"
	path = strings.Replace(path, "{"+"ChatServiceSid"+"}", ChatServiceSid, -1)
	path = strings.Replace(path, "{"+"Sid"+"}", Sid, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	resp, err := c.requestHandler.Get(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &ConversationsV1ServiceConversation{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}

// Optional parameters for the method 'ListServiceConversation'
type ListServiceConversationParams struct {
	// Start date or time in ISO8601 format for filtering list of Conversations. If a date is provided, the start time of the date is used (YYYY-MM-DDT00:00:00Z). Can be combined with other filters.
	StartDate *string `json:"StartDate,omitempty"`
	// End date or time in ISO8601 format for filtering list of Conversations. If a date is provided, the end time of the date is used (YYYY-MM-DDT23:59:59Z). Can be combined with other filters.
	EndDate *string `json:"EndDate,omitempty"`
	// State for sorting and filtering list of Conversations. Can be `active`, `inactive` or `closed`
	State *string `json:"State,omitempty"`
	// How many resources to return in each list page. The default is 50, and the maximum is 1000.
	PageSize *int `json:"PageSize,omitempty"`
	// Max number of records to return.
	Limit *int `json:"limit,omitempty"`
}

func (params *ListServiceConversationParams) SetStartDate(StartDate string) *ListServiceConversationParams {
	params.StartDate = &StartDate
	return params
}
func (params *ListServiceConversationParams) SetEndDate(EndDate string) *ListServiceConversationParams {
	params.EndDate = &EndDate
	return params
}
func (params *ListServiceConversationParams) SetState(State string) *ListServiceConversationParams {
	params.State = &State
	return params
}
func (params *ListServiceConversationParams) SetPageSize(PageSize int) *ListServiceConversationParams {
	params.PageSize = &PageSize
	return params
}
func (params *ListServiceConversationParams) SetLimit(Limit int) *ListServiceConversationParams {
	params.Limit = &Limit
	return params
}

// Retrieve a single page of ServiceConversation records from the API. Request is executed immediately.
func (c *ApiService) PageServiceConversation(ChatServiceSid string, params *ListServiceConversationParams, pageToken, pageNumber string) (*ListServiceConversationResponse, error) {
	path := "/v1/Services/{ChatServiceSid}/Conversations"

	path = strings.Replace(path, "{"+"ChatServiceSid"+"}", ChatServiceSid, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.StartDate != nil {
		data.Set("StartDate", *params.StartDate)
	}
	if params != nil && params.EndDate != nil {
		data.Set("EndDate", *params.EndDate)
	}
	if params != nil && params.State != nil {
		data.Set("State", *params.State)
	}
	if params != nil && params.PageSize != nil {
		data.Set("PageSize", fmt.Sprint(*params.PageSize))
	}

	if pageToken != "" {
		data.Set("PageToken", pageToken)
	}
	if pageNumber != "" {
		data.Set("Page", pageNumber)
	}

	resp, err := c.requestHandler.Get(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &ListServiceConversationResponse{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}

// Lists ServiceConversation records from the API as a list. Unlike stream, this operation is eager and loads 'limit' records into memory before returning.
func (c *ApiService) ListServiceConversation(ChatServiceSid string, params *ListServiceConversationParams) ([]ConversationsV1ServiceConversation, error) {
	response, errors := c.StreamServiceConversation(ChatServiceSid, params)

	records := make([]ConversationsV1ServiceConversation, 0)
	for record := range response {
		records = append(records, record)
	}

	if err := <-errors; err != nil {
		return nil, err
	}

	return records, nil
}

// Streams ServiceConversation records from the API as a channel stream. This operation lazily loads records as efficiently as possible until the limit is reached.
func (c *ApiService) StreamServiceConversation(ChatServiceSid string, params *ListServiceConversationParams) (chan ConversationsV1ServiceConversation, chan error) {
	if params == nil {
		params = &ListServiceConversationParams{}
	}
	params.SetPageSize(client.ReadLimits(params.PageSize, params.Limit))

	recordChannel := make(chan ConversationsV1ServiceConversation, 1)
	errorChannel := make(chan error, 1)

	response, err := c.PageServiceConversation(ChatServiceSid, params, "", "")
	if err != nil {
		errorChannel <- err
		close(recordChannel)
		close(errorChannel)
	} else {
		go c.streamServiceConversation(response, params, recordChannel, errorChannel)
	}

	return recordChannel, errorChannel
}

func (c *ApiService) streamServiceConversation(response *ListServiceConversationResponse, params *ListServiceConversationParams, recordChannel chan ConversationsV1ServiceConversation, errorChannel chan error) {
	curRecord := 1

	for response != nil {
		responseRecords := response.Conversations
		for item := range responseRecords {
			recordChannel <- responseRecords[item]
			curRecord += 1
			if params.Limit != nil && *params.Limit < curRecord {
				close(recordChannel)
				close(errorChannel)
				return
			}
		}

		record, err := client.GetNext(c.baseURL, response, c.getNextListServiceConversationResponse)
		if err != nil {
			errorChannel <- err
			break
		} else if record == nil {
			break
		}

		response = record.(*ListServiceConversationResponse)
	}

	close(recordChannel)
	close(errorChannel)
}

func (c *ApiService) getNextListServiceConversationResponse(nextPageUrl string) (interface{}, error) {
	if nextPageUrl == "" {
		return nil, nil
	}
	resp, err := c.requestHandler.Get(nextPageUrl, nil, nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &ListServiceConversationResponse{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}
	return ps, nil
}

// Optional parameters for the method 'UpdateServiceConversation'
type UpdateServiceConversationParams struct {
	// The X-Twilio-Webhook-Enabled HTTP request header
	XTwilioWebhookEnabled *string `json:"X-Twilio-Webhook-Enabled,omitempty"`
	// The human-readable name of this conversation, limited to 256 characters. Optional.
	FriendlyName *string `json:"FriendlyName,omitempty"`
	// The date that this resource was created.
	DateCreated *time.Time `json:"DateCreated,omitempty"`
	// The date that this resource was last updated.
	DateUpdated *time.Time `json:"DateUpdated,omitempty"`
	// An optional string metadata field you can use to store any data you wish. The string value must contain structurally valid JSON if specified.  **Note** that if the attributes are not set \\\"{}\\\" will be returned.
	Attributes *string `json:"Attributes,omitempty"`
	// The unique ID of the [Messaging Service](https://www.twilio.com/docs/messaging/api/service-resource) this conversation belongs to.
	MessagingServiceSid *string `json:"MessagingServiceSid,omitempty"`
	//
	State *string `json:"State,omitempty"`
	// ISO8601 duration when conversation will be switched to `inactive` state. Minimum value for this timer is 1 minute.
	TimersInactive *string `json:"Timers.Inactive,omitempty"`
	// ISO8601 duration when conversation will be switched to `closed` state. Minimum value for this timer is 10 minutes.
	TimersClosed *string `json:"Timers.Closed,omitempty"`
	// An application-defined string that uniquely identifies the resource. It can be used to address the resource in place of the resource's `sid` in the URL.
	UniqueName *string `json:"UniqueName,omitempty"`
	// The default email address that will be used when sending outbound emails in this conversation.
	BindingsEmailAddress *string `json:"Bindings.Email.Address,omitempty"`
	// The default name that will be used when sending outbound emails in this conversation.
	BindingsEmailName *string `json:"Bindings.Email.Name,omitempty"`
}

func (params *UpdateServiceConversationParams) SetXTwilioWebhookEnabled(XTwilioWebhookEnabled string) *UpdateServiceConversationParams {
	params.XTwilioWebhookEnabled = &XTwilioWebhookEnabled
	return params
}
func (params *UpdateServiceConversationParams) SetFriendlyName(FriendlyName string) *UpdateServiceConversationParams {
	params.FriendlyName = &FriendlyName
	return params
}
func (params *UpdateServiceConversationParams) SetDateCreated(DateCreated time.Time) *UpdateServiceConversationParams {
	params.DateCreated = &DateCreated
	return params
}
func (params *UpdateServiceConversationParams) SetDateUpdated(DateUpdated time.Time) *UpdateServiceConversationParams {
	params.DateUpdated = &DateUpdated
	return params
}
func (params *UpdateServiceConversationParams) SetAttributes(Attributes string) *UpdateServiceConversationParams {
	params.Attributes = &Attributes
	return params
}
func (params *UpdateServiceConversationParams) SetMessagingServiceSid(MessagingServiceSid string) *UpdateServiceConversationParams {
	params.MessagingServiceSid = &MessagingServiceSid
	return params
}
func (params *UpdateServiceConversationParams) SetState(State string) *UpdateServiceConversationParams {
	params.State = &State
	return params
}
func (params *UpdateServiceConversationParams) SetTimersInactive(TimersInactive string) *UpdateServiceConversationParams {
	params.TimersInactive = &TimersInactive
	return params
}
func (params *UpdateServiceConversationParams) SetTimersClosed(TimersClosed string) *UpdateServiceConversationParams {
	params.TimersClosed = &TimersClosed
	return params
}
func (params *UpdateServiceConversationParams) SetUniqueName(UniqueName string) *UpdateServiceConversationParams {
	params.UniqueName = &UniqueName
	return params
}
func (params *UpdateServiceConversationParams) SetBindingsEmailAddress(BindingsEmailAddress string) *UpdateServiceConversationParams {
	params.BindingsEmailAddress = &BindingsEmailAddress
	return params
}
func (params *UpdateServiceConversationParams) SetBindingsEmailName(BindingsEmailName string) *UpdateServiceConversationParams {
	params.BindingsEmailName = &BindingsEmailName
	return params
}

// Update an existing conversation in your service
func (c *ApiService) UpdateServiceConversation(ChatServiceSid string, Sid string, params *UpdateServiceConversationParams) (*ConversationsV1ServiceConversation, error) {
	path := "/v1/Services/{ChatServiceSid}/Conversations/{Sid}"
	path = strings.Replace(path, "{"+"ChatServiceSid"+"}", ChatServiceSid, -1)
	path = strings.Replace(path, "{"+"Sid"+"}", Sid, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.FriendlyName != nil {
		data.Set("FriendlyName", *params.FriendlyName)
	}
	if params != nil && params.DateCreated != nil {
		data.Set("DateCreated", fmt.Sprint((*params.DateCreated).Format(time.RFC3339)))
	}
	if params != nil && params.DateUpdated != nil {
		data.Set("DateUpdated", fmt.Sprint((*params.DateUpdated).Format(time.RFC3339)))
	}
	if params != nil && params.Attributes != nil {
		data.Set("Attributes", *params.Attributes)
	}
	if params != nil && params.MessagingServiceSid != nil {
		data.Set("MessagingServiceSid", *params.MessagingServiceSid)
	}
	if params != nil && params.State != nil {
		data.Set("State", *params.State)
	}
	if params != nil && params.TimersInactive != nil {
		data.Set("Timers.Inactive", *params.TimersInactive)
	}
	if params != nil && params.TimersClosed != nil {
		data.Set("Timers.Closed", *params.TimersClosed)
	}
	if params != nil && params.UniqueName != nil {
		data.Set("UniqueName", *params.UniqueName)
	}
	if params != nil && params.BindingsEmailAddress != nil {
		data.Set("Bindings.Email.Address", *params.BindingsEmailAddress)
	}
	if params != nil && params.BindingsEmailName != nil {
		data.Set("Bindings.Email.Name", *params.BindingsEmailName)
	}

	if params != nil && params.XTwilioWebhookEnabled != nil {
		headers["X-Twilio-Webhook-Enabled"] = *params.XTwilioWebhookEnabled
	}

	resp, err := c.requestHandler.Post(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &ConversationsV1ServiceConversation{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}
