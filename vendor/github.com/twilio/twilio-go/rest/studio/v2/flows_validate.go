/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Studio
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

import (
	"encoding/json"
	"net/url"
)

// Optional parameters for the method 'UpdateFlowValidate'
type UpdateFlowValidateParams struct {
	// The string that you assigned to describe the Flow.
	FriendlyName *string `json:"FriendlyName,omitempty"`
	//
	Status *string `json:"Status,omitempty"`
	// JSON representation of flow definition.
	Definition *interface{} `json:"Definition,omitempty"`
	// Description of change made in the revision.
	CommitMessage *string `json:"CommitMessage,omitempty"`
}

func (params *UpdateFlowValidateParams) SetFriendlyName(FriendlyName string) *UpdateFlowValidateParams {
	params.FriendlyName = &FriendlyName
	return params
}
func (params *UpdateFlowValidateParams) SetStatus(Status string) *UpdateFlowValidateParams {
	params.Status = &Status
	return params
}
func (params *UpdateFlowValidateParams) SetDefinition(Definition interface{}) *UpdateFlowValidateParams {
	params.Definition = &Definition
	return params
}
func (params *UpdateFlowValidateParams) SetCommitMessage(CommitMessage string) *UpdateFlowValidateParams {
	params.CommitMessage = &CommitMessage
	return params
}

// Validate flow JSON definition
func (c *ApiService) UpdateFlowValidate(params *UpdateFlowValidateParams) (*StudioV2FlowValidate, error) {
	path := "/v2/Flows/Validate"

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.FriendlyName != nil {
		data.Set("FriendlyName", *params.FriendlyName)
	}
	if params != nil && params.Status != nil {
		data.Set("Status", *params.Status)
	}
	if params != nil && params.Definition != nil {
		v, err := json.Marshal(params.Definition)

		if err != nil {
			return nil, err
		}

		data.Set("Definition", string(v))
	}
	if params != nil && params.CommitMessage != nil {
		data.Set("CommitMessage", *params.CommitMessage)
	}

	resp, err := c.requestHandler.Post(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &StudioV2FlowValidate{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}
