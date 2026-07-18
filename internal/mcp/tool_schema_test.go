package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
)

// Claude (Anthropic) rejects tool input_schema that is not valid JSON Schema draft 2020-12.
// A common footgun: objectProperty(desc, propertiesMap) puts the map under additionalProperties,
// so a nested field named "type" becomes an invalid schema "type" keyword value.
func TestToolInputSchemasClaudeCompatible(t *testing.T) {
	server := &Server{}
	for i, def := range server.toolDefinitions() {
		name := def.tool.Name
		if err := validateJSONSchemaDraft202012(def.tool.InputSchema); err != nil {
			t.Errorf("tools[%d] %s: invalid input_schema: %v", i, name, err)
		}
		// Must be JSON-serializable (MCP wire format).
		if _, err := json.Marshal(def.tool.InputSchema); err != nil {
			t.Errorf("tools[%d] %s: schema not JSON-serializable: %v", i, name, err)
		}
	}
}

func TestCreateProjectBuildSettingsIsNestedObject(t *testing.T) {
	server := &Server{}
	var create Tool
	for _, def := range server.toolDefinitions() {
		if def.tool.Name == "create_project" {
			create = def.tool
			break
		}
	}
	if create.Name == "" {
		t.Fatal("create_project tool not found")
	}
	props, ok := create.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("create_project properties missing")
	}
	bs, ok := props["build_settings"].(map[string]interface{})
	if !ok {
		t.Fatal("build_settings property missing")
	}
	if bs["type"] != "object" {
		t.Fatalf("build_settings.type = %v, want object", bs["type"])
	}
	// Named fields must live under properties, not additionalProperties.
	nestedProps, ok := bs["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("build_settings.properties missing; schema = %#v", bs)
	}
	if _, ok := nestedProps["build_method"]; !ok {
		t.Fatal("expected build_method in build_settings.properties")
	}
	if ap, ok := bs["additionalProperties"]; ok {
		// free-form bool is fine; a property map here is the Claude-breaking bug
		if _, isMap := ap.(map[string]interface{}); isMap {
			t.Fatalf("build_settings.additionalProperties must not be a property map: %#v", ap)
		}
	}
}

func validateJSONSchemaDraft202012(schema map[string]interface{}) error {
	return validateSchemaNode(schema, "$")
}

func validateSchemaNode(node interface{}, path string) error {
	switch n := node.(type) {
	case map[string]interface{}:
		if typeVal, ok := n["type"]; ok {
			switch tv := typeVal.(type) {
			case string:
				// ok
			case []interface{}:
				for i, item := range tv {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("%s.type[%d] must be string, got %T", path, i, item)
					}
				}
			default:
				return fmt.Errorf("%s.type must be string or array of strings, got %T (%v)", path, typeVal, typeVal)
			}
		}
		if props, ok := n["properties"].(map[string]interface{}); ok {
			for name, child := range props {
				if err := validateSchemaNode(child, path+".properties."+name); err != nil {
					return err
				}
			}
		}
		if items, ok := n["items"]; ok {
			if err := validateSchemaNode(items, path+".items"); err != nil {
				return err
			}
		}
		if ap, ok := n["additionalProperties"]; ok {
			switch ap.(type) {
			case bool:
				// ok
			case map[string]interface{}:
				if err := validateSchemaNode(ap, path+".additionalProperties"); err != nil {
					return err
				}
			default:
				return fmt.Errorf("%s.additionalProperties must be bool or schema object, got %T", path, ap)
			}
		}
		if required, ok := n["required"]; ok {
			arr, ok := required.([]string)
			if !ok {
				// objectSchema uses []string; also accept []interface{}
				if iface, ok := required.([]interface{}); ok {
					for i, item := range iface {
						if _, ok := item.(string); !ok {
							return fmt.Errorf("%s.required[%d] must be string", path, i)
						}
					}
				} else {
					return fmt.Errorf("%s.required must be array of strings, got %T", path, required)
				}
			} else {
				_ = arr
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: expected object schema, got %T", path, node)
	}
}
