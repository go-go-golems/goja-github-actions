package contextdata

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
)

func LoadEventPayload(path string) (map[string]interface{}, error) {
	if strings.TrimSpace(path) == "" {
		return map[string]interface{}{}, nil
	}

	payloadBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "read event payload %s", path)
	}

	payload := map[string]interface{}{}
	if len(payloadBytes) == 0 {
		return payload, nil
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.Wrap(err, "decode event payload")
	}

	return payload, nil
}
