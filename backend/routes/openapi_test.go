package routes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
)

// These checks cover draft contract consistency; implementation flags describe
// NewContextHandler's HTTP layer, not database readiness or adapter acceptance.
func TestContextOpenAPIOperations(t *testing.T) {
	doc := readContextOpenAPI(t)
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
	if doc["x-relio-contract-status"] != "draft-awaiting-o1" {
		t.Errorf("contract status = %v, want draft-awaiting-o1", doc["x-relio-contract-status"])
	}

	paths := contextOpenAPIObject(t, doc, doc["paths"])
	implementations := map[string]string{
		"/healthz":                      "implemented",
		"/api/deals":                    "implemented",
		"/api/deals/{deal_id}":          "implemented",
		"/api/deals/{deal_id}/graph":    "implemented",
		"/api/deals/{deal_id}/timeline": "implemented",
		"/api/evidence/{evidence_id}":   "implemented",
	}
	if len(paths) != len(implementations) {
		t.Errorf("path count = %d, want %d", len(paths), len(implementations))
	}
	for path := range paths {
		if _, ok := implementations[path]; !ok {
			t.Errorf("unexpected path %q", path)
		}
	}

	operationIDs := map[string]string{}
	for path, implementation := range implementations {
		t.Run(path, func(t *testing.T) {
			item := contextOpenAPIObject(t, doc, paths[path])
			operation := contextOpenAPIObject(t, doc, item["get"])
			id, _ := operation["operationId"].(string)
			if strings.TrimSpace(id) == "" {
				t.Error("GET operationId is missing or empty")
			} else if previous, ok := operationIDs[id]; ok {
				t.Errorf("operationId %q also used by %s", id, previous)
			} else {
				operationIDs[id] = path
			}
			if operation["x-relio-implementation"] != implementation {
				t.Errorf("implementation = %v, want %s", operation["x-relio-implementation"], implementation)
			}

			responses := contextOpenAPIObject(t, doc, operation["responses"])
			response := contextOpenAPIObject(t, doc, responses["200"])
			content := contextOpenAPIObject(t, doc, response["content"])
			media := contextOpenAPIObject(t, doc, content["application/json"])
			if schema := contextOpenAPIObject(t, doc, media["schema"]); len(schema) == 0 {
				t.Error("200 application/json schema is empty")
			}

			parameters := contextOpenAPIParameters(t, doc, item, operation)
			for _, segment := range strings.Split(path, "/") {
				if strings.HasPrefix(segment, "{") {
					name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
					if parameter := parameters["path:"+name]; parameter == nil || parameter["required"] != true {
						t.Errorf("path parameter %q must be present and required", name)
					}
				}
			}
			if strings.HasPrefix(path, "/api/") {
				parameter := contextOpenAPIObject(t, doc, parameters["query:as_of"])
				schema := contextOpenAPIObject(t, doc, parameter["schema"])
				if schema["format"] != "date" || schema["default"] != "2026-10-01" {
					t.Errorf("as_of schema = %v, want format date and default 2026-10-01", schema)
				}
			}

			bounds := map[string][2]float64{}
			switch path {
			case "/api/deals/{deal_id}/graph":
				bounds = map[string][2]float64{"depth": {1, 2}, "max_nodes": {150, 150}, "max_edges": {300, 300}}
			case "/api/deals", "/api/deals/{deal_id}/timeline":
				if _, exposed := parameters["query:limit"]; exposed {
					bounds["limit"] = [2]float64{50, 50}
				}
			}
			for name, want := range bounds {
				parameter := contextOpenAPIObject(t, doc, parameters["query:"+name])
				schema := contextOpenAPIObject(t, doc, parameter["schema"])
				if schema["minimum"] != float64(1) || schema["default"] != want[0] || schema["maximum"] != want[1] {
					t.Errorf("%s minimum/default/maximum = %v/%v/%v, want 1/%v/%v", name, schema["minimum"], schema["default"], schema["maximum"], want[0], want[1])
				}
			}
		})
	}
}

func TestContextOpenAPISchemas(t *testing.T) {
	doc := readContextOpenAPI(t)
	components := contextOpenAPIObject(t, doc, doc["components"])
	schemas := contextOpenAPIObject(t, doc, components["schemas"])
	factFields := []string{"value", "state", "temporal_basis", "evidence_ids", "limitations"}
	required := map[string][]string{
		"FactString":         factFields,
		"FactInteger":        factFields,
		"FactDate":           factFields,
		"SnapshotMeta":       {"context_id", "as_of", "max_as_of", "calendar_zone", "contract_version", "dataset_version", "data_state", "unknowns", "limitations"},
		"Bounds":             {"truncated", "truncation_reasons", "next_cursor"},
		"DealFacts":          {"deal_id", "account_id", "deal_type", "owner_id", "stage", "stage_since", "created_at", "planned_outlets", "potential_acv_idr", "status", "record_evidence_ids"},
		"DealListResponse":   {"meta", "bounds", "items"},
		"DealDetailResponse": {"meta", "deal"},
		"GraphResponse":      {"meta", "bounds", "nodes", "edges"},
		"TimelineResponse":   {"meta", "bounds", "events"},
		"EvidenceResponse":   {"meta", "evidence"},
		"ErrorResponse":      {"code", "message"},
		"HealthResponse":     {"status", "service"},
	}
	factTypes := map[string]string{"FactString": "string", "FactInteger": "integer", "FactDate": "string"}
	for name, fields := range required {
		t.Run(name, func(t *testing.T) {
			schema := contextOpenAPIObject(t, doc, schemas[name])
			properties := contextOpenAPIObject(t, doc, schema["properties"])
			for _, field := range fields {
				if !contextOpenAPIContains(schema["required"], field) {
					t.Errorf("required is missing %q", field)
				}
				if _, ok := properties[field]; !ok {
					t.Errorf("properties is missing %q", field)
				}
			}
			if kind, fact := factTypes[name]; fact {
				value := contextOpenAPIObject(t, doc, properties["value"])
				if !contextOpenAPIHasType(t, doc, value, kind) || !contextOpenAPIHasType(t, doc, value, "null") {
					t.Errorf("value must allow %s and null, got %v", kind, value)
				}
				state := contextOpenAPIObject(t, doc, properties["state"])
				for _, want := range []string{"known", "unknown", "ambiguous", "snapshot_only"} {
					if !contextOpenAPIContains(state["enum"], want) {
						t.Errorf("state enum is missing %q", want)
					}
				}
				if name == "FactInteger" {
					if minimum, ok := value["minimum"].(float64); ok && minimum > 0 {
						t.Errorf("value minimum %v excludes known zero", minimum)
					}
					if minimum, ok := value["exclusiveMinimum"].(float64); ok && minimum >= 0 {
						t.Errorf("value exclusiveMinimum %v excludes known zero", minimum)
					}
				}
			}
			if name == "ErrorResponse" {
				for field := range properties {
					switch strings.ToLower(field) {
					case "cause", "query", "password":
						t.Errorf("error response declares internal field %q", field)
					}
				}
			}
		})
	}
}

func TestContextOpenAPILocalReferences(t *testing.T) {
	doc := readContextOpenAPI(t)
	var walk func(any, string)
	walk = func(value any, location string) {
		switch value := value.(type) {
		case map[string]any:
			for key, child := range value {
				if key == "$ref" {
					ref, ok := child.(string)
					if !ok {
						t.Errorf("%s/$ref must be a string", location)
					} else if _, err := resolveContextOpenAPIRef(doc, ref); err != nil {
						t.Errorf("%s/$ref: %v", location, err)
					}
				}
				walk(child, location+"/"+key)
			}
		case []any:
			for index, child := range value {
				walk(child, location+"/"+strconv.Itoa(index))
			}
		}
	}
	walk(doc, "#")
}

func readContextOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("../docs/context-api.openapi.json")
	if err != nil {
		t.Fatalf("read draft OpenAPI contract: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode draft OpenAPI contract: %v", err)
	}
	if doc == nil {
		t.Fatal("draft OpenAPI contract must be a JSON object")
	}
	return doc
}

func contextOpenAPIObject(t *testing.T, doc map[string]any, value any) map[string]any {
	t.Helper()
	seen := map[string]bool{}
	for {
		object, ok := value.(map[string]any)
		if !ok || object == nil {
			t.Fatalf("want contract object, got %T", value)
		}
		raw, referenced := object["$ref"]
		if !referenced {
			return object
		}
		ref, ok := raw.(string)
		if !ok || seen[ref] {
			t.Fatalf("invalid or cyclic object reference %v", raw)
		}
		seen[ref] = true
		var err error
		value, err = resolveContextOpenAPIRef(doc, ref)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func resolveContextOpenAPIRef(doc map[string]any, ref string) (any, error) {
	if !strings.HasPrefix(ref, "#") {
		return nil, fmt.Errorf("reference %q must be local", ref)
	}
	pointer, err := url.PathUnescape(strings.TrimPrefix(ref, "#"))
	if err != nil || (pointer != "" && !strings.HasPrefix(pointer, "/")) {
		return nil, fmt.Errorf("reference %q must be a local JSON pointer", ref)
	}
	var value any = doc
	if pointer == "" {
		return value, nil
	}
	for _, token := range strings.Split(pointer[1:], "/") {
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		switch current := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = current[token]
			if !ok {
				return nil, fmt.Errorf("reference %q has missing token %q", ref, token)
			}
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(current) || strconv.Itoa(index) != token {
				return nil, fmt.Errorf("reference %q has invalid array index %q", ref, token)
			}
			value = current[index]
		default:
			return nil, fmt.Errorf("reference %q cannot traverse token %q", ref, token)
		}
	}
	return value, nil
}

func contextOpenAPIParameters(t *testing.T, doc, item, operation map[string]any) map[string]map[string]any {
	t.Helper()
	parameters := map[string]map[string]any{}
	for _, source := range []map[string]any{item, operation} {
		raw, exists := source["parameters"]
		if !exists {
			continue
		}
		list, ok := raw.([]any)
		if !ok {
			t.Fatalf("parameters must be an array, got %T", raw)
		}
		for _, value := range list {
			parameter := contextOpenAPIObject(t, doc, value)
			name, _ := parameter["name"].(string)
			location, _ := parameter["in"].(string)
			parameters[location+":"+name] = parameter
		}
	}
	return parameters
}

func contextOpenAPIContains(values any, want string) bool {
	list, _ := values.([]any)
	for _, value := range list {
		if value == want {
			return true
		}
	}
	return false
}

func contextOpenAPIHasType(t *testing.T, doc map[string]any, value any, want string) bool {
	t.Helper()
	schema := contextOpenAPIObject(t, doc, value)
	if schema["type"] == want || contextOpenAPIContains(schema["type"], want) {
		return true
	}
	for _, keyword := range []string{"anyOf", "oneOf"} {
		branches, _ := schema[keyword].([]any)
		for _, branch := range branches {
			if contextOpenAPIHasType(t, doc, branch, want) {
				return true
			}
		}
	}
	return false
}
