/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Trusthub
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
	"strings"
)

// Optional parameters for the method 'CreateComplianceTollfreeInquiry'
type CreateComplianceTollfreeInquiryParams struct {
	// The Tollfree phone number to be verified
	Did *string `json:"Did,omitempty"`
}

func (params *CreateComplianceTollfreeInquiryParams) SetDid(Did string) *CreateComplianceTollfreeInquiryParams {
	params.Did = &Did
	return params
}

// Create a new Compliance Tollfree Verification Inquiry for the authenticated account. This is necessary to start a new embedded session.
func (c *ApiService) CreateComplianceTollfreeInquiry(params *CreateComplianceTollfreeInquiryParams) (*TrusthubV1ComplianceTollfreeInquiry, error) {
	path := "/v1/ComplianceInquiries/Tollfree/Initialize"

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.Did != nil {
		data.Set("Did", *params.Did)
	}

	resp, err := c.requestHandler.Post(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &TrusthubV1ComplianceTollfreeInquiry{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}

// Optional parameters for the method 'UpdateComplianceTollfreeInquiry'
type UpdateComplianceTollfreeInquiryParams struct {
	// The Tollfree phone number to be verified
	Did *string `json:"Did,omitempty"`
}

func (params *UpdateComplianceTollfreeInquiryParams) SetDid(Did string) *UpdateComplianceTollfreeInquiryParams {
	params.Did = &Did
	return params
}

// Resume a specific Compliance Tollfree Verification Inquiry that has expired, or re-open a rejected Compliance Tollfree Verification Inquiry for editing.
func (c *ApiService) UpdateComplianceTollfreeInquiry(TollfreeId string, params *UpdateComplianceTollfreeInquiryParams) (*TrusthubV1ComplianceTollfreeInquiry, error) {
	path := "/v1/ComplianceInquiries/Tollfree/{TollfreeId}/Initialize"
	path = strings.Replace(path, "{"+"TollfreeId"+"}", TollfreeId, -1)

	data := url.Values{}
	headers := make(map[string]interface{})

	if params != nil && params.Did != nil {
		data.Set("Did", *params.Did)
	}

	resp, err := c.requestHandler.Post(c.baseURL+path, data, headers)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	ps := &TrusthubV1ComplianceTollfreeInquiry{}
	if err := json.NewDecoder(resp.Body).Decode(ps); err != nil {
		return nil, err
	}

	return ps, err
}