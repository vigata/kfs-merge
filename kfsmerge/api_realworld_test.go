package kfsmerge

import (
	"encoding/json"
	"testing"
)

// TestKfsMediaSchemaPattern tests common patterns from kfs_media schema usage.
func TestKfsMediaSchemaPattern(t *testing.T) {
	// Simplified version of kfs_media patterns with video filters
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"video_filters": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"type": {"type": "string"},
						"params": {"type": "object"}
					}
				},
				"x-kfs-merge": {"strategy": "mergeByDiscriminator", "discriminatorField": "type"}
			},
			"encoding": {
				"type": "object",
				"x-kfs-merge": {"strategy": "overlay"},
				"properties": {
					"codec": {"type": "string"},
					"bitrate": {"type": "integer"},
					"profile": {"type": "string"}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Base template has multiple filters and encoding settings
	b := []byte(`{
		"video_filters": [
			{"type": "hqdn3d", "params": {"strength": 4}},
			{"type": "unsharp", "params": {"amount": 1.5}}
		],
		"encoding": {
			"codec": "h264",
			"bitrate": 5000000,
			"profile": "high"
		}
	}`)

	// Request overrides hqdn3d and changes bitrate
	a := []byte(`{
		"video_filters": [
			{"type": "hqdn3d", "params": {"strength": 8}}
		],
		"encoding": {
			"bitrate": 8000000
		}
	}`)

	result, err := s.Merge(a, b)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Check video_filters
	filters := got["video_filters"].([]any)
	if len(filters) != 2 {
		t.Fatalf("video_filters length = %d, want 2", len(filters))
	}

	for _, f := range filters {
		filter := f.(map[string]any)
		if filter["type"] == "hqdn3d" {
			params := filter["params"].(map[string]any)
			if params["strength"] != float64(8) {
				t.Errorf("hqdn3d strength = %v, want 8", params["strength"])
			}
		}
	}

	// Check encoding - overlay should preserve codec and profile
	encoding := got["encoding"].(map[string]any)
	if encoding["codec"] != "h264" {
		t.Errorf("encoding.codec = %v, want 'h264'", encoding["codec"])
	}
	if encoding["bitrate"] != float64(8000000) {
		t.Errorf("encoding.bitrate = %v, want 8000000", encoding["bitrate"])
	}
	if encoding["profile"] != "high" {
		t.Errorf("encoding.profile = %v, want 'high'", encoding["profile"])
	}
}

// TestForcedKeyframesOptionReplace tests replace strategy for mutually exclusive option objects.
func TestForcedKeyframesOptionReplace(t *testing.T) {
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"forced_keyframes": {
				"type": "object",
				"x-kfs-merge": {"strategy": "replace"},
				"properties": {
					"keyframes_timecodes": {
						"anyOf": [
							{"type": "array", "items": {"type": "string"}},
							{"type": "null"}
						]
					},
					"keyframes_frame_numbers": {
						"anyOf": [
							{"type": "array", "items": {"type": "integer"}},
							{"type": "null"}
						]
					},
					"keyframes_in_s": {
						"anyOf": [
							{"type": "array", "items": {"type": "number"}},
							{"type": "null"}
						]
					}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	tests := []struct {
		name                  string
		base                  string
		request               string
		wantTimecodes         []string
		wantFrameNumbers      []float64
		wantInSeconds         []float64
		expectTimecodesAbsent bool
		expectFrameNumsAbsent bool
		expectInSecondsAbsent bool
	}{
		{
			name:             "replace timecodes with frame_numbers",
			base:             `{"forced_keyframes": {"keyframes_timecodes": ["00:00:10:00"], "keyframes_frame_numbers": null}}`,
			request:          `{"forced_keyframes": {"keyframes_timecodes": null, "keyframes_frame_numbers": [100, 200]}}`,
			wantTimecodes:    nil,
			wantFrameNumbers: []float64{100, 200},
		},
		{
			name:                  "request with partial fields replaces entirely",
			base:                  `{"forced_keyframes": {"keyframes_timecodes": ["00:00:05:00"], "keyframes_frame_numbers": [50]}}`,
			request:               `{"forced_keyframes": {"keyframes_frame_numbers": [999]}}`,
			wantFrameNumbers:      []float64{999},
			expectTimecodesAbsent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.Merge([]byte(tt.request), []byte(tt.base))
			if err != nil {
				t.Fatalf("Merge failed: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			fk := got["forced_keyframes"].(map[string]any)

			// Check keyframes_timecodes
			if tt.expectTimecodesAbsent {
				if _, exists := fk["keyframes_timecodes"]; exists {
					t.Errorf("keyframes_timecodes should be absent")
				}
			} else if tt.wantTimecodes == nil {
				if fk["keyframes_timecodes"] != nil {
					t.Errorf("keyframes_timecodes = %v, want nil", fk["keyframes_timecodes"])
				}
			}

			// Check keyframes_frame_numbers
			if tt.wantFrameNumbers != nil {
				frameNums := fk["keyframes_frame_numbers"].([]any)
				if len(frameNums) != len(tt.wantFrameNumbers) {
					t.Errorf("keyframes_frame_numbers length = %d, want %d", len(frameNums), len(tt.wantFrameNumbers))
				}
			}
		})
	}
}

// TestForcedKeyframesOptionWithoutReplace shows default deepMerge behavior (contrast with replace).
func TestForcedKeyframesOptionWithoutReplace(t *testing.T) {
	// Without replace strategy, deepMerge would deep merge, potentially leaving mixed state.
	schemaJSON := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"forced_keyframes": {
				"type": "object",
				"properties": {
					"keyframes_timecodes": {
						"anyOf": [
							{"type": "array", "items": {"type": "string"}},
							{"type": "null"}
						]
					},
					"keyframes_frame_numbers": {
						"anyOf": [
							{"type": "array", "items": {"type": "integer"}},
							{"type": "null"}
						]
					}
				}
			}
		}
	}`)

	s, err := LoadSchema(schemaJSON)
	if err != nil {
		t.Fatalf("LoadSchema failed: %v", err)
	}

	// Base has timecodes, request has frame_numbers (with partial fields)
	base := `{"forced_keyframes": {"keyframes_timecodes": ["00:00:10:00"], "keyframes_frame_numbers": null}}`
	request := `{"forced_keyframes": {"keyframes_frame_numbers": [100, 200]}}`

	result, err := s.Merge([]byte(request), []byte(base))
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(result, &got); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	fk := got["forced_keyframes"].(map[string]any)

	// With default deepMerge, both fields would be present (B's timecodes + A's frame_numbers)
	// This is the "mixed state" that replace strategy avoids

	// keyframes_timecodes should be preserved from base: ["00:00:10:00"]
	timecodes := fk["keyframes_timecodes"].([]any)
	if len(timecodes) != 1 || timecodes[0].(string) != "00:00:10:00" {
		t.Errorf("keyframes_timecodes = %v, want [\"00:00:10:00\"] (preserved from base)", timecodes)
	}

	// keyframes_frame_numbers should come from request: [100, 200]
	frameNums := fk["keyframes_frame_numbers"].([]any)
	if len(frameNums) != 2 {
		t.Fatalf("keyframes_frame_numbers length = %d, want 2", len(frameNums))
	}
	if frameNums[0].(float64) != 100 || frameNums[1].(float64) != 200 {
		t.Errorf("keyframes_frame_numbers = %v, want [100, 200]", frameNums)
	}
}
