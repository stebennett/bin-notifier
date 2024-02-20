/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Api
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

// ApiV2010SipCredentialList struct for ApiV2010SipCredentialList
type ApiV2010SipCredentialList struct {
	// The unique id of the [Account](https://www.twilio.com/docs/iam/api/account) that owns this resource.
	AccountSid *string `json:"account_sid,omitempty"`
	// The date that this resource was created, given as GMT in [RFC 2822](https://www.php.net/manual/en/class.datetime.php#datetime.constants.rfc2822) format.
	DateCreated *string `json:"date_created,omitempty"`
	// The date that this resource was last updated, given as GMT in [RFC 2822](https://www.php.net/manual/en/class.datetime.php#datetime.constants.rfc2822) format.
	DateUpdated *string `json:"date_updated,omitempty"`
	// A human readable descriptive text that describes the CredentialList, up to 64 characters long.
	FriendlyName *string `json:"friendly_name,omitempty"`
	// A 34 character string that uniquely identifies this resource.
	Sid *string `json:"sid,omitempty"`
	// A list of credentials associated with this credential list.
	SubresourceUris *map[string]interface{} `json:"subresource_uris,omitempty"`
	// The URI for this resource, relative to `https://api.twilio.com`.
	Uri *string `json:"uri,omitempty"`
}
