package memory

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
)

func CloneMessages(messages []anthropic.MessageParam) ([]anthropic.MessageParam, error) {

	data, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	var cloned []anthropic.MessageParam

	err = json.Unmarshal(data, &cloned)
	if err != nil {
		return nil, err
	}

	return cloned, nil
}
