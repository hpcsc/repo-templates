package credential

import (
	"fmt"
	"net/url"
	"os"
)

const (
	apiURL         = ""
	apiURLVariable = "GO_CLI_PROJECT_API_URL"
)

func APIURL() (*url.URL, error) {
	value := setting(apiURLVariable, apiURL)
	if value == "" {
		return nil, fmt.Errorf("the API URL is not set: set apiURL in internal/credential/api_url.go, or set %s", apiURLVariable)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("the API URL %q: %w", value, err)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("the API URL %q has no host", value)
	}
	return parsed, nil
}

func APIHost() (string, error) {
	parsed, err := APIURL()
	if err != nil {
		return "", err
	}
	return parsed.Host, nil
}

func setting(variable, constant string) string {
	if value := os.Getenv(variable); value != "" {
		return value
	}
	return constant
}
