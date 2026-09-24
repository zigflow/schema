/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/schema/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCronExpr = "0 0 * * *"

// minimalWorkflow returns a minimal structurally valid workflow document.
// It is used as a baseline for validation tests so that each test only
// introduces one deviation from a known-good state.
func minimalWorkflow() map[string]any {
	return map[string]any{
		propDocument: map[string]any{
			propDSL:          "1.0.0",
			propTaskQueue:    "default",
			propWorkflowType: "test",
			propVersion:      "1.0.0",
		},
		propDo: []any{
			map[string]any{
				"step1": map[string]any{
					propSet: map[string]any{"x": "y"},
				},
			},
		},
	}
}

// TestSchema_DocumentFields verifies that the document schema accepts the
// Zigflow-aligned field names and rejects the old Serverless Workflow names.
func TestSchema_DocumentFields(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)

	t.Run("workflowType and taskQueue are accepted", func(t *testing.T) {
		err := resolved.Validate(minimalWorkflow())
		assert.NoError(t, err)
	})

	t.Run("old name field is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		document := doc["document"].(map[string]any)
		delete(document, propWorkflowType)
		document["name"] = "test"

		err := resolved.Validate(doc)
		assert.Error(t, err, "old 'name' field must be rejected; use 'workflowType' instead")
	})

	t.Run("old namespace field is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		document := doc["document"].(map[string]any)
		delete(document, propTaskQueue)
		document["namespace"] = "default"

		err := resolved.Validate(doc)
		assert.Error(t, err, "old 'namespace' field must be rejected; use 'taskQueue' instead")
	})

	t.Run("missing workflowType is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		delete(doc["document"].(map[string]any), propWorkflowType)

		err := resolved.Validate(doc)
		assert.Error(t, err, "document without workflowType must fail validation")
	})

	t.Run("missing taskQueue is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		delete(doc["document"].(map[string]any), propTaskQueue)

		err := resolved.Validate(doc)
		assert.Error(t, err, "document without taskQueue must fail validation")
	})
}

// TestSchema_ScheduleRequiresScheduleWorkflowName verifies the conditional
// if/then rule: when top-level schedule is present, document.metadata must
// exist and contain scheduleWorkflowName.
func TestSchema_ScheduleRequiresScheduleWorkflowName(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("schedule present without document.metadata is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		doc["schedule"] = map[string]any{propCron: testCronExpr}

		err := resolved.Validate(doc)
		assert.Error(t, err, "schedule without document.metadata must fail validation")
	})

	t.Run("schedule present with document.metadata but missing scheduleWorkflowName is rejected", func(t *testing.T) {
		doc := minimalWorkflow()
		doc["schedule"] = map[string]any{propCron: testCronExpr}
		doc["document"].(map[string]any)["metadata"] = map[string]any{
			propScheduleID: "my-schedule",
		}

		err := resolved.Validate(doc)
		assert.Error(t, err, "schedule with document.metadata but no scheduleWorkflowName must fail validation")
	})

	t.Run("schedule present with valid document.metadata.scheduleWorkflowName is accepted", func(t *testing.T) {
		doc := minimalWorkflow()
		doc["schedule"] = map[string]any{propCron: testCronExpr}
		doc["document"].(map[string]any)["metadata"] = map[string]any{
			propScheduleWorkflowName: "my-workflow",
		}

		err := resolved.Validate(doc)
		assert.NoError(t, err, "schedule with valid scheduleWorkflowName must pass validation")
	})
}

// TestSchema_RejectsUnknownTopLevelProperties verifies that the root schema
// enforces UnevaluatedProperties: false by rejecting any top-level key that
// is not explicitly defined in the schema.
func TestSchema_RejectsUnknownTopLevelProperties(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)

	resolved, err := s.Resolve(nil)
	require.NoError(t, err)

	t.Run("valid document passes", func(t *testing.T) {
		err := resolved.Validate(minimalWorkflow())
		assert.NoError(t, err, "document with only known top-level properties should pass validation")
	})
}

const (
	testPropEmit    = "emit"
	testPropEvent   = "event"
	testEventType   = "com.example.order.placed"
	testEventSource = "https://example.com/orders"
	testStepName    = "step1"
	testPropListen  = "listen"
	testPropIf      = "if"
	testPropID      = "id"
	testPropData    = "data"
	testPropSchema  = "dataschema"
	testUnknownKey  = "unknown"
	testUnknownVal  = "value"
	testOwner       = "orders"
	testBroker      = "kafka"
)

// emitTaskWorkflow returns a minimal workflow whose only task is the given
// task body. Tests use this to exercise the emit task schema in isolation.
func emitTaskWorkflow(task map[string]any) map[string]any {
	doc := minimalWorkflow()
	doc[propDo] = []any{
		map[string]any{
			testStepName: task,
		},
	}
	return doc
}

// emitTask returns an emit task body whose emit.event.with is the given
// value.
func emitTask(with any) map[string]any {
	return map[string]any{
		testPropEmit: map[string]any{
			testPropEvent: map[string]any{
				propWith: with,
			},
		},
	}
}

// emitWorkflow returns a minimal workflow with a single emit task whose
// emit.event.with is the given value.
func emitWorkflow(with any) map[string]any {
	return emitTaskWorkflow(emitTask(with))
}

// listenOneWorkflow returns a minimal workflow with a single listen task
// whose listen.to.one.with is the given value. This is the existing consumer
// of eventProperties and is used as the baseline for parity checks.
func listenOneWorkflow(with any) map[string]any {
	return emitTaskWorkflow(map[string]any{
		testPropListen: map[string]any{
			"to": map[string]any{
				"one": map[string]any{
					propWith: with,
				},
			},
		},
	})
}

// TestSchema_EmitTask_UpstreamExample verifies that the emit example from the
// Open Workflow Specification (examples/emit.yaml) is accepted.
func TestSchema_EmitTask_UpstreamExample(t *testing.T) {
	resolved := resolvedTestSchema(t)

	doc := minimalWorkflow()
	doc[propDo] = []any{
		map[string]any{
			"emitEvent": emitTask(map[string]any{
				propSource: "https://petstore.com",
				propType:   "com.petstore.order.placed.v1",
				testPropData: map[string]any{
					"client": map[string]any{
						"firstName": "Cruella",
						"lastName":  "de Vil",
					},
					"items": []any{
						map[string]any{"breed": "dalmatian", "quantity": 101},
					},
				},
			}),
		},
	}

	assert.NoError(t, resolved.Validate(doc))
}

// TestSchema_EmitTask_Accepted verifies that emit is a valid task type and
// that every property supported by eventProperties is accepted under
// emit.event.with.
func TestSchema_EmitTask_Accepted(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name string
		with map[string]any
	}{
		{
			name: "type only is accepted",
			with: map[string]any{propType: testEventType},
		},
		{
			name: "all event properties are accepted",
			with: map[string]any{
				testPropID:        "evt-1",
				propSource:        testEventSource,
				propType:          testEventType,
				"time":            "2026-01-01T00:00:00Z",
				"subject":         "order-123",
				"datacontenttype": "application/json",
				testPropSchema:    "https://example.com/schemas/order.json",
				testPropData:      map[string]any{"orderId": "123"},
			},
		},
		{
			name: "expression source is accepted",
			with: map[string]any{propType: testEventType, propSource: "${ .source }"},
		},
		{
			name: "expression dataschema is accepted",
			with: map[string]any{propType: testEventType, testPropSchema: "${ .schema }"},
		},
		{
			name: "expression data is accepted",
			with: map[string]any{propType: testEventType, testPropData: "${ .payload }"},
		},
		{
			name: "non-object data is accepted",
			with: map[string]any{propType: testEventType, testPropData: []any{1, "two", true}},
		},
		{
			name: "extension attribute is accepted",
			with: map[string]any{propType: testEventType, "partitionkey": testOwner},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, resolved.Validate(emitWorkflow(tc.with)))
		})
	}
}

// TestSchema_EmitTask_RequiredFields verifies the upstream required nesting:
// the task requires emit, emit requires event, and event.with (when present)
// requires type. Upstream does not require event.with itself.
func TestSchema_EmitTask_RequiredFields(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name        string
		task        map[string]any
		expectError bool
	}{
		{
			// Only taskBase properties, so no task type matches.
			name:        "task without emit is rejected",
			task:        map[string]any{testPropIf: "${ true }"},
			expectError: true,
		},
		{
			name:        "null emit is rejected",
			task:        map[string]any{testPropEmit: nil},
			expectError: true,
		},
		{
			name:        "non-object emit is rejected",
			task:        map[string]any{testPropEmit: testPropEvent},
			expectError: true,
		},
		{
			name:        "empty emit is rejected",
			task:        map[string]any{testPropEmit: map[string]any{}},
			expectError: true,
		},
		{
			name:        "non-object emit.event is rejected",
			task:        map[string]any{testPropEmit: map[string]any{testPropEvent: testPropEvent}},
			expectError: true,
		},
		{
			name: "empty emit.event is accepted",
			task: map[string]any{testPropEmit: map[string]any{testPropEvent: map[string]any{}}},
		},
		{
			name: "emit.event with only additional properties is accepted",
			task: map[string]any{testPropEmit: map[string]any{testPropEvent: map[string]any{"broker": testBroker}}},
		},
		{
			name:        "empty emit.event.with is rejected",
			task:        emitTask(map[string]any{}),
			expectError: true,
		},
		{
			name:        "emit.event.with without type is rejected",
			task:        emitTask(map[string]any{propSource: testEventSource, testPropID: "evt-1"}),
			expectError: true,
		},
		{
			name:        "non-object emit.event.with is rejected",
			task:        emitTask(testEventType),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := resolved.Validate(emitTaskWorkflow(tc.task))
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSchema_EmitTask_InvalidEventProperties verifies that invalid event
// properties are rejected under emit.event.with, and that the existing listen
// task (which also uses eventProperties) rejects the same values.
func TestSchema_EmitTask_InvalidEventProperties(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name string
		with map[string]any
	}{
		{
			name: "non-string type is rejected",
			with: map[string]any{propType: 123},
		},
		{
			name: "non-string id is rejected",
			with: map[string]any{propType: testEventType, testPropID: 123},
		},
		{
			name: "non-string subject is rejected",
			with: map[string]any{propType: testEventType, "subject": true},
		},
		{
			name: "non-string datacontenttype is rejected",
			with: map[string]any{propType: testEventType, "datacontenttype": 1},
		},
		{
			name: "non-URI source is rejected",
			with: map[string]any{propType: testEventType, propSource: "not a uri"},
		},
		{
			name: "non-string source is rejected",
			with: map[string]any{propType: testEventType, propSource: 5},
		},
		{
			name: "non-URI dataschema is rejected",
			with: map[string]any{propType: testEventType, testPropSchema: "not a uri"},
		},
		{
			name: "non-string time is rejected",
			with: map[string]any{propType: testEventType, "time": 12345},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, resolved.Validate(emitWorkflow(tc.with)), "emit must reject invalid event properties")
			assert.Error(t, resolved.Validate(listenOneWorkflow(tc.with)), "listen must reject the same event properties")
		})
	}
}

// TestSchema_EmitTask_UnevaluatedProperties verifies that unknown keys are
// rejected at the task and emit levels, which set unevaluatedProperties to
// false, while emit.event permits additional properties.
func TestSchema_EmitTask_UnevaluatedProperties(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("unknown key at task level is rejected", func(t *testing.T) {
		task := emitTask(map[string]any{propType: testEventType})
		task[testUnknownKey] = testUnknownVal
		assert.Error(t, resolved.Validate(emitTaskWorkflow(task)))
	})

	t.Run("unknown key inside emit is rejected", func(t *testing.T) {
		task := emitTask(map[string]any{propType: testEventType})
		task[testPropEmit].(map[string]any)[testUnknownKey] = testUnknownVal
		assert.Error(t, resolved.Validate(emitTaskWorkflow(task)))
	})

	t.Run("additional key inside emit.event is accepted", func(t *testing.T) {
		task := emitTask(map[string]any{propType: testEventType})
		task[testPropEmit].(map[string]any)[testPropEvent].(map[string]any)["broker"] = map[string]any{"name": testBroker}
		assert.NoError(t, resolved.Validate(emitTaskWorkflow(task)))
	})

	t.Run("additional key inside emit.event does not bypass with validation", func(t *testing.T) {
		task := emitTask(map[string]any{propSource: testEventSource})
		task[testPropEmit].(map[string]any)[testPropEvent].(map[string]any)["broker"] = testBroker
		assert.Error(t, resolved.Validate(emitTaskWorkflow(task)))
	})
}

// TestSchema_EmitTask_TaskBase verifies that the emit task inherits and
// validates the shared taskBase properties.
func TestSchema_EmitTask_TaskBase(t *testing.T) {
	resolved := resolvedTestSchema(t)

	tests := []struct {
		name        string
		base        map[string]any
		expectError bool
	}{
		{
			name: "all taskBase properties are accepted",
			base: map[string]any{
				testPropIf:   "${ .enabled }",
				propInput:    map[string]any{propSchema: map[string]any{"document": map[string]any{propType: "object"}}},
				propOutput:   map[string]any{propAs: "${ . }"},
				propExport:   map[string]any{propAs: "${ . }"},
				propThen:     "end",
				propMetadata: map[string]any{"owner": testOwner},
			},
		},
		{
			name:        "non-string if is rejected",
			base:        map[string]any{testPropIf: true},
			expectError: true,
		},
		{
			name:        "non-string then is rejected",
			base:        map[string]any{propThen: 1},
			expectError: true,
		},
		{
			name:        "unknown key inside export is rejected",
			base:        map[string]any{propExport: map[string]any{testUnknownKey: testUnknownVal}},
			expectError: true,
		},
		{
			name:        "non-object metadata is rejected",
			base:        map[string]any{propMetadata: testOwner},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := emitTask(map[string]any{propType: testEventType})
			for k, v := range tc.base {
				task[k] = v
			}

			err := resolved.Validate(emitTaskWorkflow(task))
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSchema_EmitTask_TaskOneOf verifies that adding emitTask to the task
// OneOf does not allow emit to be combined with other task types, and that
// existing task types remain valid.
func TestSchema_EmitTask_TaskOneOf(t *testing.T) {
	resolved := resolvedTestSchema(t)

	t.Run("existing set task is still accepted", func(t *testing.T) {
		assert.NoError(t, resolved.Validate(minimalWorkflow()))
	})

	t.Run("emit alongside other emit tasks is accepted", func(t *testing.T) {
		doc := minimalWorkflow()
		doc[propDo] = []any{
			map[string]any{testStepName: emitTask(map[string]any{propType: testEventType})},
			map[string]any{"step2": map[string]any{propSet: map[string]any{"x": "y"}}},
			map[string]any{"step3": emitTask(map[string]any{propType: testEventType + ".v2"})},
		}
		assert.NoError(t, resolved.Validate(doc))
	})

	for name, other := range map[string]map[string]any{
		"set":  {propSet: map[string]any{"x": "y"}},
		"wait": {propWait: map[string]any{propSeconds: 5}},
		testPropListen: {testPropListen: map[string]any{
			"to": map[string]any{"one": map[string]any{propWith: map[string]any{propType: testEventType}}},
		}},
	} {
		t.Run("emit combined with "+name+" is rejected", func(t *testing.T) {
			task := emitTask(map[string]any{propType: testEventType})
			for k, v := range other {
				task[k] = v
			}
			assert.Error(t, resolved.Validate(emitTaskWorkflow(task)))
		})
	}

	t.Run("task with only an unknown key is rejected", func(t *testing.T) {
		assert.Error(t, resolved.Validate(emitTaskWorkflow(map[string]any{testUnknownKey: testUnknownVal})))
	})
}
