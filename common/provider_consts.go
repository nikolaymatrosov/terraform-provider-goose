package common

const (
	DefaultMaxRetries = 5
)

var Descriptions = map[string]string{
	"endpoint": "The API endpoint for Yandex.Cloud SDK client.",

	"token": "The access token for API operations.",

	"service_account_key_file": "Either the path to or the contents of a Service Account key file in JSON format.",
	"max_retries": "The maximum number of times an API request is being executed. \n" +
		"If the API request still fails, an error is thrown.",
}
