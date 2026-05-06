package bt

import "fmt"

// BuildTreeFromYAML recursively constructs a Node from parsed YAML data.
// Each element is either:
//   - a string (leaf name, delegated to resolve)
//   - a map with one key (composite type: "sequence", "selector", "reactive_selector")
//
// The resolve function maps leaf name strings to concrete Nodes. It must handle
// any domain-specific logic such as "!" inversion, parameterized leaves, and subtrees.
func BuildTreeFromYAML(data any, resolve func(string) (Node, error)) (Node, error) {
	switch v := data.(type) {
	case string:
		return resolve(v)

	case map[string]any:
		if len(v) != 1 {
			return nil, fmt.Errorf("tree node map must have exactly one key, got %d", len(v))
		}
		for key, val := range v {
			children, ok := val.([]any)
			if !ok {
				return nil, fmt.Errorf("composite %q: children must be a list", key)
			}
			nodes := make([]Node, 0, len(children))
			for i, child := range children {
				n, err := BuildTreeFromYAML(child, resolve)
				if err != nil {
					return nil, fmt.Errorf("composite %q child %d: %w", key, i, err)
				}
				nodes = append(nodes, n)
			}
			switch key {
			case "sequence":
				return NewSequence(nodes...), nil
			case "selector":
				return NewSelector(nodes...), nil
			case "reactive_selector":
				return NewReactiveSelector(nodes...), nil
			default:
				return nil, fmt.Errorf("unknown composite type: %q", key)
			}
		}
	}

	return nil, fmt.Errorf("unexpected tree node type %T", data)
}
