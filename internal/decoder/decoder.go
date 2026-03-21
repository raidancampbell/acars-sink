package decoder

import "encoding/json"

// Message represents a minimal ACARS JSON payload with room to grow.
// Unknown fields are retained in Raw for future enrichment.
type Message struct {
	Aircraft     string         `json:"aircraft"`
	Flight       string         `json:"flight"`
	Type         string         `json:"type"`
	Station      string         `json:"station"`
	Timestamp    StringOrNumber `json:"timestamp,omitempty"`
	Label        string         `json:"label,omitempty"`
	Message      string         `json:"message,omitempty"`
	Text         string         `json:"text,omitempty"`
	Channel      StringOrNumber `json:"channel,omitempty"`
	Registration string         `json:"registration,omitempty"`
	ICAO         string         `json:"icao,omitempty"`
	Raw          map[string]any `json:"-"`
}

type messageAlias Message

// StringOrNumber allows JSON values that may be encoded as string or number.
type StringOrNumber string

func (s *StringOrNumber) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*s = ""
		return nil
	}
	if data[0] == '"' {
		var val string
		if err := json.Unmarshal(data, &val); err != nil {
			return err
		}
		*s = StringOrNumber(val)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = StringOrNumber(num.String())
	return nil
}

func (m *Message) UnmarshalJSON(data []byte) error {
	var alias messageAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	delete(raw, "aircraft")
	delete(raw, "flight")
	delete(raw, "type")
	delete(raw, "station")
	delete(raw, "timestamp")
	delete(raw, "label")
	delete(raw, "message")
	delete(raw, "text")
	delete(raw, "channel")
	delete(raw, "registration")
	delete(raw, "icao")

	*m = Message(alias)
	m.Raw = raw
	return nil
}
