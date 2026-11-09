package network

import (
	"encoding/json"
	"log"
)

// mustMarshal serializes v to JSON and logs on failure.
// Panics on error — use only for payloads that are known to be marshalable.
func MustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Failed to marshal: %v", err)
		return nil
	}
	return data
}
