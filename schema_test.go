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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCronExpr     = "0 0 * * *"
	testWorkflowType = "my-workflow"
)

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

func runWorkflowDoc(workflow map[string]any) map[string]any {
	doc := minimalWorkflow()
	doc[propDo] = []any{
		map[string]any{
			"callChild": map[string]any{
				"run": map[string]any{
					"workflow": workflow,
				},
			},
		},
	}
	return doc
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

// TestSchema_ScheduleNeedsNoWorkflowKey verifies that a schedule validates
// without a workflow key: the runtime defaults it to document.workflowType, and
// only it can tell when that does not apply (every top-level task a Do task).
func TestSchema_ScheduleNeedsNoWorkflowKey(t *testing.T) {
	resolved := resolvedTestSchema(t)

	for name, meta := range map[string]map[string]any{
		"no document.metadata":   nil,
		"metadata without a key": {propScheduleID: "my-schedule"},
		"the workflow type key":  {propScheduleWorkflowType: testWorkflowType},
		"the deprecated alias":   {propScheduleWorkflowName: testWorkflowType},
	} {
		t.Run(name, func(t *testing.T) {
			doc := minimalWorkflow()
			doc["schedule"] = map[string]any{propCron: testCronExpr}
			if meta != nil {
				doc["document"].(map[string]any)["metadata"] = meta
			}

			assert.NoError(t, resolved.Validate(doc), "a schedule must validate with %s", name)
		})
	}
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

func TestSchema_RunWorkflowTaskQueue(t *testing.T) {
	resolved := resolvedTestSchema(t)
	tests := []struct {
		name       string
		workflow   map[string]any
		awaitFalse bool
		wantError  bool
	}{
		{
			name:     "without taskQueue is accepted",
			workflow: map[string]any{propType: testWorkflowType},
		},
		{
			name:     "with taskQueue is accepted",
			workflow: map[string]any{propType: testWorkflowType, propTaskQueue: "other-queue"},
		},
		{
			name:     "orders.v2 is accepted",
			workflow: map[string]any{propType: testWorkflowType, propTaskQueue: "orders.v2"},
		},
		{
			name:     "test_workflow is accepted",
			workflow: map[string]any{propType: testWorkflowType, propTaskQueue: "test_workflow"},
		},
		{
			name:     "single-character name is accepted",
			workflow: map[string]any{propType: testWorkflowType, propTaskQueue: "a"},
		},
		{
			name: "64-character name is accepted",
			workflow: map[string]any{
				propType:      testWorkflowType,
				propTaskQueue: strings.Repeat("a", 64),
			},
		},
		{
			name: "taskQueue with input and await false is accepted",
			workflow: map[string]any{
				propType:      testWorkflowType,
				propTaskQueue: "other-queue",
				propInput:     map[string]any{"a": 1},
			},
			awaitFalse: true,
		},
		{
			name: "empty taskQueue is rejected",
			workflow: map[string]any{
				propType:      testWorkflowType,
				propTaskQueue: "",
			},
			wantError: true,
		},
		{
			name: "integer taskQueue is rejected",
			workflow: map[string]any{
				propType:      testWorkflowType,
				propTaskQueue: 123,
			},
			wantError: true,
		},
		{
			name: "object taskQueue is rejected",
			workflow: map[string]any{
				propType:      testWorkflowType,
				propTaskQueue: map[string]any{},
			},
			wantError: true,
		},
		{
			name: "unknown sibling property is rejected",
			workflow: map[string]any{
				propType: testWorkflowType,
				"queue":  "x",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := runWorkflowDoc(tt.workflow)
			if tt.awaitFalse {
				task := doc[propDo].([]any)[0].(map[string]any)["callChild"].(map[string]any)
				task["run"].(map[string]any)["await"] = false
			}

			err := resolved.Validate(doc)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
