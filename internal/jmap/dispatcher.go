package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunt/internal/domain"
	"yunt/internal/jmap/core"
)

// MethodFunc is the handler signature for JMAP method implementations.
type MethodFunc func(ctx context.Context, accountID domain.ID, args json.RawMessage) (json.RawMessage, *core.MethodError)

// Dispatcher routes JMAP method calls to registered handlers.
type Dispatcher struct {
	methods  map[string]MethodFunc
	maxCalls int
}

// NewDispatcher creates a new JMAP method dispatcher.
func NewDispatcher(maxCalls int) *Dispatcher {
	if maxCalls <= 0 {
		maxCalls = 64
	}
	return &Dispatcher{
		methods:  make(map[string]MethodFunc),
		maxCalls: maxCalls,
	}
}

// Register adds a method handler.
func (d *Dispatcher) Register(name string, fn MethodFunc) {
	d.methods[name] = fn
}

// Dispatch processes a JMAP request and returns a response.
func (d *Dispatcher) Dispatch(ctx context.Context, accountID domain.ID, req *core.Request) *core.Response {
	resp := &core.Response{
		MethodResponses: make([]core.Invocation, 0, len(req.MethodCalls)),
		CreatedIds:      req.CreatedIds,
	}

	if len(req.MethodCalls) > d.maxCalls {
		resp.MethodResponses = append(resp.MethodResponses,
			core.NewMethodError(core.ErrorRequestTooLarge,
				fmt.Sprintf("too many method calls: %d > %d", len(req.MethodCalls), d.maxCalls)).
				ToInvocation(""))
		return resp
	}

	results := make(map[string]json.RawMessage)

	for _, call := range req.MethodCalls {
		args, err := d.resolveResultRefs(call.Args, results)
		if err != nil {
			resp.MethodResponses = append(resp.MethodResponses,
				core.NewMethodError(core.ErrorInvalidResultReference, err.Error()).ToInvocation(call.CallID))
			continue
		}

		fn, ok := d.methods[call.Name]
		if !ok {
			resp.MethodResponses = append(resp.MethodResponses,
				core.NewMethodError(core.ErrorUnknownMethod, call.Name).ToInvocation(call.CallID))
			continue
		}

		result, methodErr := fn(ctx, accountID, args)
		if methodErr != nil {
			resp.MethodResponses = append(resp.MethodResponses, methodErr.ToInvocation(call.CallID))
			continue
		}

		results[call.CallID] = result
		resp.MethodResponses = append(resp.MethodResponses, core.Invocation{
			Name:   call.Name,
			Args:   result,
			CallID: call.CallID,
		})
	}

	return resp
}

// resolveResultRefs substitutes #property references in args with values from previous results.
func (d *Dispatcher) resolveResultRefs(args json.RawMessage, results map[string]json.RawMessage) (json.RawMessage, error) {
	var argsMap map[string]json.RawMessage
	if err := json.Unmarshal(args, &argsMap); err != nil {
		return args, nil
	}

	resolved := false
	for key, val := range argsMap {
		if !strings.HasPrefix(key, "#") {
			continue
		}

		targetKey := key[1:]
		if _, hasPlain := argsMap[targetKey]; hasPlain {
			return nil, fmt.Errorf("both %q and %q present", key, targetKey)
		}

		var ref core.ResultReference
		if err := json.Unmarshal(val, &ref); err != nil {
			return nil, fmt.Errorf("invalid result reference for %q: %w", key, err)
		}

		prevResult, ok := results[ref.ResultOf]
		if !ok {
			return nil, fmt.Errorf("resultOf %q not found", ref.ResultOf)
		}

		extracted, err := extractPath(prevResult, ref.Path)
		if err != nil {
			return nil, fmt.Errorf("path %q extraction failed: %w", ref.Path, err)
		}

		delete(argsMap, key)
		argsMap[targetKey] = extracted
		resolved = true
	}

	if !resolved {
		return args, nil
	}

	return json.Marshal(argsMap)
}

// extractPath extracts a value from JSON using a JSON Pointer path with JMAP * wildcard extension.
func extractPath(data json.RawMessage, path string) (json.RawMessage, error) {
	if path == "" || path == "/" {
		return data, nil
	}

	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	return extractPathParts(data, parts)
}

func extractPathParts(data json.RawMessage, parts []string) (json.RawMessage, error) {
	if len(parts) == 0 {
		return data, nil
	}

	key := parts[0]
	rest := parts[1:]

	if key == "*" {
		var arr []json.RawMessage
		if err := json.Unmarshal(data, &arr); err != nil {
			return nil, fmt.Errorf("* applied to non-array")
		}

		var collected []json.RawMessage
		for _, elem := range arr {
			val, err := extractPathParts(elem, rest)
			if err != nil {
				return nil, err
			}
			var subArr []json.RawMessage
			if json.Unmarshal(val, &subArr) == nil {
				collected = append(collected, subArr...)
			} else {
				collected = append(collected, val)
			}
		}
		return json.Marshal(collected)
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("cannot index into non-object with key %q", key)
	}

	val, ok := obj[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}

	return extractPathParts(val, rest)
}
