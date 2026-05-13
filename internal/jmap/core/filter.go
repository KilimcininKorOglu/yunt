package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FilterNode represents either a FilterOperator (AND/OR/NOT) or a FilterCondition.
type FilterNode struct {
	Operator   string           `json:"operator,omitempty"`
	Conditions []json.RawMessage `json:"conditions,omitempty"`

	// Leaf fields are present when this is a condition, not an operator
	IsLeaf    bool
	Condition json.RawMessage
}

// UnmarshalJSON handles both operator nodes and leaf conditions.
func (f *FilterNode) UnmarshalJSON(data []byte) error {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}

	if _, hasOp := probe["operator"]; hasOp {
		var op struct {
			Operator   string            `json:"operator"`
			Conditions []json.RawMessage `json:"conditions"`
		}
		if err := json.Unmarshal(data, &op); err != nil {
			return err
		}
		f.Operator = op.Operator
		f.Conditions = op.Conditions
		f.IsLeaf = false
		return nil
	}

	f.IsLeaf = true
	f.Condition = data
	return nil
}

// BuildSQLWhere converts a JMAP filter tree to SQL WHERE clause fragments.
// Returns (clause string, args []interface{}).
// fieldMapper converts JMAP filter property names to SQL column expressions.
func BuildSQLWhere(filter json.RawMessage, fieldMapper func(key string, value interface{}) (string, interface{}, error), maxDepth int) (string, []interface{}, error) {
	if filter == nil || string(filter) == "null" || string(filter) == "{}" {
		return "1=1", nil, nil
	}
	return buildNode(filter, fieldMapper, 0, maxDepth)
}

func buildNode(data json.RawMessage, fieldMapper func(string, interface{}) (string, interface{}, error), depth, maxDepth int) (string, []interface{}, error) {
	if depth > maxDepth {
		return "", nil, fmt.Errorf("filter depth exceeds maximum of %d", maxDepth)
	}

	var node FilterNode
	if err := json.Unmarshal(data, &node); err != nil {
		return "", nil, err
	}

	if node.IsLeaf {
		return buildLeaf(node.Condition, fieldMapper)
	}

	if len(node.Conditions) == 0 {
		return "1=1", nil, nil
	}

	op := strings.ToUpper(node.Operator)
	if op != "AND" && op != "OR" && op != "NOT" {
		return "", nil, fmt.Errorf("unknown filter operator: %s", node.Operator)
	}

	var clauses []string
	var allArgs []interface{}

	for _, cond := range node.Conditions {
		clause, args, err := buildNode(cond, fieldMapper, depth+1, maxDepth)
		if err != nil {
			return "", nil, err
		}
		clauses = append(clauses, clause)
		allArgs = append(allArgs, args...)
	}

	if op == "NOT" {
		combined := strings.Join(clauses, " AND ")
		return fmt.Sprintf("NOT (%s)", combined), allArgs, nil
	}

	combined := strings.Join(clauses, fmt.Sprintf(" %s ", op))
	return fmt.Sprintf("(%s)", combined), allArgs, nil
}

func buildLeaf(data json.RawMessage, fieldMapper func(string, interface{}) (string, interface{}, error)) (string, []interface{}, error) {
	var props map[string]interface{}
	if err := json.Unmarshal(data, &props); err != nil {
		return "", nil, err
	}

	var clauses []string
	var args []interface{}

	for key, value := range props {
		clause, arg, err := fieldMapper(key, value)
		if err != nil {
			continue
		}
		clauses = append(clauses, clause)
		if arg != nil {
			args = append(args, arg)
		}
	}

	if len(clauses) == 0 {
		return "1=1", nil, nil
	}

	return strings.Join(clauses, " AND "), args, nil
}
