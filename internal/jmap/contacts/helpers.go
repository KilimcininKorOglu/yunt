package contacts

import (
	"encoding/json"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
)

func marshalJSON(v interface{}) (json.RawMessage, *core.MethodError) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, core.NewMethodError(core.ErrorServerFail, err.Error())
	}
	return data, nil
}

func marshalGetResponse(accountID, stateStr string, list []map[string]interface{}, notFound []string) (json.RawMessage, *core.MethodError) {
	if list == nil {
		list = []map[string]interface{}{}
	}
	if notFound == nil {
		notFound = []string{}
	}
	return marshalJSON(map[string]interface{}{
		"accountId": accountID,
		"state":     stateStr,
		"list":      list,
		"notFound":  notFound,
	})
}

func idSlice(ids []domain.ID) []string {
	if ids == nil {
		return []string{}
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}
