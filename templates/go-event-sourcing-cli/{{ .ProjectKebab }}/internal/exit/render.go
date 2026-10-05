package exit

import (
	"encoding/json"
	"io"
)

func WriteResult(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func WriteError(w io.Writer, err error) error {
	classified := classify(err)
	body := struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}{
		Error: classified.Message,
		Code:  classified.Label,
	}
	return json.NewEncoder(w).Encode(body)
}
