/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Microvisor
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

import (
	"time"
)

// MicrovisorV1DeviceConfig struct for MicrovisorV1DeviceConfig
type MicrovisorV1DeviceConfig struct {
	// A 34-character string that uniquely identifies the parent Device.
	DeviceSid *string `json:"device_sid,omitempty"`
	// The config key; up to 100 characters.
	Key *string `json:"key,omitempty"`
	// The config value; up to 4096 characters.
	Value       *string    `json:"value,omitempty"`
	DateUpdated *time.Time `json:"date_updated,omitempty"`
	// The absolute URL of the Config.
	Url *string `json:"url,omitempty"`
}
