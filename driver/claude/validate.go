package claude

import "github.com/dpopsuev/djinn/driver"

// validateRequest checks that the driver state is valid before sending
// a request to the API. Returns a ValidationError if something is wrong.
func (d *APIDriver) validateRequest(messages []apiMessage) error {
	if len(messages) == 0 {
		return &driver.ValidationError{
			Field:   "messages",
			Message: "at least one message is required",
		}
	}

	if d.apiURL == "" {
		return &driver.ValidationError{
			Field:   "api_url",
			Message: "API URL is not configured",
		}
	}

	if d.apiKey == "" {
		return &driver.ValidationError{
			Field:   "api_key",
			Message: "API key or access token is not set",
		}
	}

	// Direct API requires model in request body.
	if !d.useVertex && d.resolveModel() == "" {
		return &driver.ValidationError{
			Field:   "model",
			Message: "model is required for Claude Direct API",
		}
	}

	return nil
}
