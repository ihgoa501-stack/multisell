package supplyevent

import "encoding/json"

// ToPayload converts a typed event struct to map[string]interface{} for
// publishing via the EventBus.
func ToPayload(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
