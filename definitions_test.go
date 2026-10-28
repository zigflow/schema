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
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -- helpers ------------------------------------------------------------------

// schemaRefs collects all $ref strings from a slice of schemas.
func schemaRefs(schemas []*jsonschema.Schema) []string {
	refs := make([]string, 0, len(schemas))
	for _, s := range schemas {
		refs = append(refs, s.Ref)
	}
	return refs
}

// callConst returns the value of the "call" const from a callTask OneOf entry.
// Each entry is an object with AllOf[1].Properties["call"].Const.
func callConst(s *jsonschema.Schema) string {
	if len(s.AllOf) < 2 {
		return ""
	}
	props := s.AllOf[1].Properties
	if props == nil {
		return ""
	}
	call, ok := props["call"]
	if !ok || call.Const == nil {
		return ""
	}
	v, ok := (*call.Const).(string)
	if !ok {
		return ""
	}
	return v
}

// runVariantTitles returns the Title fields from the OneOf list inside
// runTaskDefinition's "run" property.
func runVariantTitles() []string {
	run := runTaskDefinition.AllOf[1].Properties["run"]
	titles := make([]string, 0, len(run.OneOf))
	for _, v := range run.OneOf {
		titles = append(titles, v.Title)
	}
	return titles
}

// catchProperty returns the named property from tryTask's catch sub-schema,
// navigating AllOf[1].Properties["catch"].Properties[name].
func catchProperty(name string) *jsonschema.Schema {
	catch := tryTaskDefinition.AllOf[1].Properties["catch"]
	if catch == nil || catch.Properties == nil {
		return nil
	}
	return catch.Properties[name]
}

// -- tests --------------------------------------------------------------------

// TestBuildSchema verifies the schema builds successfully and resolves without
// errors. This is the top-level smoke test.
func TestBuildSchema(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)
	require.NotNil(t, s)
}

// TestBuildDefinitionsKeys verifies that buildDefinitions returns exactly the
// expected set of supported definitions and that intentionally unsupported
// definitions are absent.
func TestBuildDefinitionsKeys(t *testing.T) {
	defs := buildDefinitions()

	expected := []string{
		"callTask",
		defCommonMetadata,
		"containerLifetime",
		"doTask",
		defDocumentMetadata,
		"duration",
		"emitTask",
		propEndpoint,
		propError,
		"eventConsumptionStrategy",
		"eventFilter",
		"eventProperties",
		propExport,
		"externalResource",
		"flowDirective",
		"forkTask",
		"forTask",
		propInput,
		"listenTask",
		defOutput,
		"raiseTask",
		"runTask",
		"runtimeExpression",
		defSchema,
		"setTask",
		"subscriptionIterator",
		"switchTask",
		"task",
		"taskBase",
		"taskList",
		defTaskMetadata,
		defTimeout,
		"tryTask",
		"uriTemplate",
		"waitTask",
	}

	for _, key := range expected {
		assert.Contains(t, defs, key, "buildDefinitions() should contain %q", key)
	}
}

// TestTaskDefinitionOneOf verifies that taskDefinition references each of the
// supported task types.
func TestTaskDefinitionOneOf(t *testing.T) {
	refs := schemaRefs(taskDefinition.OneOf)

	supported := []string{
		SchemaRef("callTask"),
		SchemaRef("doTask"),
		SchemaRef("emitTask"),
		SchemaRef("forTask"),
		SchemaRef("forkTask"),
		SchemaRef("listenTask"),
		SchemaRef("raiseTask"),
		SchemaRef("runTask"),
		SchemaRef("setTask"),
		SchemaRef("switchTask"),
		SchemaRef("tryTask"),
		SchemaRef("waitTask"),
	}

	for _, ref := range supported {
		assert.Contains(t, refs, ref, "taskDefinition.OneOf should reference %q", ref)
	}

	assert.Len(t, refs, len(supported), "taskDefinition.OneOf must not reference any unexpected task types")
}

// TestEmitTaskDefinitionShape verifies that emitTask mirrors the upstream Open
// Workflow Specification definition: it inherits taskBase, closes the task and
// emit levels to unknown keys, requires emit, emit.event and (when present)
// emit.event.with.type, and reuses the shared eventProperties definition.
func TestEmitTaskDefinitionShape(t *testing.T) {
	e := emitTaskDefinition

	assert.Equal(t, typeObject, e.Type)
	assert.Equal(t, []string{"emit"}, e.Required, "emitTask must require 'emit'")
	assert.Equal(t, falseSchema(), e.UnevaluatedProperties, "emitTask must reject unevaluated properties")

	require.Len(t, e.AllOf, 2)
	assert.Equal(t, SchemaRef("taskBase"), e.AllOf[0].Ref, "emitTask must inherit taskBase")

	emit, ok := e.AllOf[1].Properties["emit"]
	require.True(t, ok, "emit property must be present")
	assert.Equal(t, typeObject, emit.Type)
	assert.Equal(t, []string{testPropEvent}, emit.Required, "emit must require 'event'")
	assert.Equal(t, falseSchema(), emit.UnevaluatedProperties, "emit must reject unevaluated properties")

	event, ok := emit.Properties[testPropEvent]
	require.True(t, ok, "emit.event property must be present")
	assert.Equal(t, typeObject, event.Type)
	assert.Empty(t, event.Required, "emit.event must not require any properties (upstream leaves 'with' optional)")
	assert.Equal(t, trueSchema(), event.AdditionalProperties, "emit.event must allow additional properties")

	with, ok := event.Properties[propWith]
	require.True(t, ok, "emit.event.with property must be present")
	assert.Equal(t, SchemaRef("eventProperties"), with.Ref, "emit.event.with must reference eventProperties")
	assert.Equal(t, []string{propType}, with.Required, "emit.event.with must require 'type'")
}

// TestEmitTaskDefinitionJSON verifies that emitTask serialises to the upstream
// JSON Schema keywords within the built schema, in particular that the
// falseSchema/trueSchema helpers marshal to boolean schemas and that emitTask
// is part of the task union.
func TestEmitTaskDefinitionJSON(t *testing.T) {
	s, err := BuildSchema("1.0.0", "json")
	require.NoError(t, err)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var doc struct {
		Defs map[string]json.RawMessage `json:"$defs"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))

	var task struct {
		OneOf []*jsonschema.Schema `json:"oneOf"`
	}
	require.NoError(t, json.Unmarshal(doc.Defs["task"], &task))
	assert.Contains(t, schemaRefs(task.OneOf), SchemaRef("emitTask"), "$defs.task.oneOf must reference emitTask")

	type withJSON struct {
		Ref      string   `json:"$ref"`
		Required []string `json:"required"`
	}
	type eventJSON struct {
		Type                 string              `json:"type"`
		Required             []string            `json:"required"`
		AdditionalProperties any                 `json:"additionalProperties"`
		Properties           map[string]withJSON `json:"properties"`
	}
	type emitJSON struct {
		Type                  string               `json:"type"`
		Required              []string             `json:"required"`
		UnevaluatedProperties any                  `json:"unevaluatedProperties"`
		Properties            map[string]eventJSON `json:"properties"`
	}
	var emitTaskJSON struct {
		Type                  string   `json:"type"`
		Required              []string `json:"required"`
		UnevaluatedProperties any      `json:"unevaluatedProperties"`
		AllOf                 []struct {
			Ref        string              `json:"$ref"`
			Properties map[string]emitJSON `json:"properties"`
		} `json:"allOf"`
	}
	require.NoError(t, json.Unmarshal(doc.Defs["emitTask"], &emitTaskJSON))

	assert.Equal(t, typeObject, emitTaskJSON.Type)
	assert.Equal(t, []string{"emit"}, emitTaskJSON.Required)
	assert.Equal(t, false, emitTaskJSON.UnevaluatedProperties, "emitTask.unevaluatedProperties must marshal to false")
	require.Len(t, emitTaskJSON.AllOf, 2)
	assert.Equal(t, SchemaRef("taskBase"), emitTaskJSON.AllOf[0].Ref)

	emit := emitTaskJSON.AllOf[1].Properties["emit"]
	assert.Equal(t, typeObject, emit.Type)
	assert.Equal(t, []string{testPropEvent}, emit.Required)
	assert.Equal(t, false, emit.UnevaluatedProperties, "emit.unevaluatedProperties must marshal to false")

	event := emit.Properties[testPropEvent]
	assert.Equal(t, typeObject, event.Type)
	assert.Empty(t, event.Required, "emit.event must not declare required properties")
	assert.Equal(t, true, event.AdditionalProperties, "emit.event.additionalProperties must marshal to true")

	with := event.Properties[propWith]
	assert.Equal(t, SchemaRef("eventProperties"), with.Ref)
	assert.Equal(t, []string{propType}, with.Required)
}

// TestDurationDefinitionShape verifies that duration supports only the object
// form and does not include the runtimeExpression or ISO 8601 string branches
// that exist in the upstream Serverless Workflow schema.
func TestDurationDefinitionShape(t *testing.T) {
	d := durationDefinition

	assert.Len(t, d.OneOf, 1, "duration must have exactly one OneOf branch (object form only)")

	branch := d.OneOf[0]
	assert.Equal(t, typeObject, branch.Type, "the single duration branch must be an object")

	for _, prop := range []string{propDays, propHours, propMinutes, propSeconds, propMilliseconds} {
		assert.Contains(t, branch.Properties, prop, "duration object must have property %q", prop)
	}

	// Confirm the two unsupported branches are absent.
	for _, s := range d.OneOf {
		assert.Empty(t, s.Ref, "duration must not contain a $ref branch (no runtimeExpression)")
		assert.Empty(t, s.Pattern, "duration must not contain a pattern branch (no ISO 8601 literal)")
	}
}

// TestWaitTaskDefinitionShape verifies that waitTaskDefinition's wait property
// is a OneOf of the duration-with-expressions form and the until form, and
// that the shared duration definition is not used (so other consumers of
// durationDefinition are not affected by the wait task extensions).
func TestWaitTaskDefinitionShape(t *testing.T) {
	require.Len(t, waitTaskDefinition.AllOf, 2)

	waitProp, ok := waitTaskDefinition.AllOf[1].Properties[propWait]
	require.True(t, ok, "wait property must be present")
	require.Empty(t, waitProp.Ref, "wait property must not reference the shared duration definition")
	require.Len(t, waitProp.OneOf, 2, "wait property must be a OneOf with exactly two branches")

	// First branch: duration-with-expressions object form.
	duration := waitProp.OneOf[0]
	assert.Equal(t, typeObject, duration.Type, "first branch must be the duration object form")
	for _, prop := range []string{propDays, propHours, propMinutes, propSeconds, propMilliseconds} {
		field, ok := duration.Properties[prop]
		require.True(t, ok, "duration form must have property %q", prop)
		require.Len(t, field.OneOf, 2, "duration field %q must be a OneOf with two branches", prop)
		assert.Equal(t, typeInteger, field.OneOf[0].Type, "duration field %q first branch must be integer", prop)
		assert.Equal(t, SchemaRef("runtimeExpression"), field.OneOf[1].Ref, "duration field %q second branch must be runtimeExpression", prop)
	}

	// Second branch: until-only object form.
	until := waitProp.OneOf[1]
	assert.Equal(t, typeObject, until.Type, "second branch must be the until object form")
	assert.Contains(t, until.Required, propUntil, "until branch must require 'until'")
	untilField, ok := until.Properties[propUntil]
	require.True(t, ok, "until branch must have an 'until' property")
	require.Len(t, untilField.OneOf, 2, "until property must be a OneOf with two branches")
	assert.Equal(t, rfc3339Pattern, untilField.OneOf[0].Pattern, "until first branch must enforce RFC 3339 via pattern")
	assert.Equal(t, SchemaRef("runtimeExpression"), untilField.OneOf[1].Ref, "until second branch must be runtimeExpression")
}

// TestTryTaskDefinitionCatch verifies that tryTask requires both "try" and
// "catch", and that catch requires "do" referencing taskList.
func TestTryTaskDefinitionCatch(t *testing.T) {
	assert.Contains(t, tryTaskDefinition.Required, "try")
	assert.Contains(t, tryTaskDefinition.Required, "catch")

	catch := tryTaskDefinition.AllOf[1].Properties["catch"]
	require.NotNil(t, catch, "catch property must be present")

	assert.Equal(t, typeObject, catch.Type)
	assert.Contains(t, catch.Required, "do")

	doField := catchProperty("do")
	require.NotNil(t, doField, "catch.do must be present")
	assert.Equal(t, SchemaRef("taskList"), doField.Ref, "catch.do must reference taskList")
}

// TestCallTaskDefinitionVariants verifies that callTaskDefinition contains
// exactly the supported call variants and not the unsupported ones.
func TestCallTaskDefinitionVariants(t *testing.T) {
	consts := make([]string, 0, len(callTaskDefinition.OneOf))
	for _, s := range callTaskDefinition.OneOf {
		consts = append(consts, callConst(s))
	}

	for _, want := range []string{"activity", "grpc", "http"} {
		assert.True(t, slices.Contains(consts, want), "callTask should support %q", want)
	}

	for _, absent := range []string{"asyncapi", "openapi", "mcp", "a2a"} {
		assert.False(t, slices.Contains(consts, absent), "callTask must not support %q", absent)
	}
}

// TestRunTaskDefinitionVariants verifies that runTaskDefinition includes all
// four supported run modes.
func TestRunTaskDefinitionVariants(t *testing.T) {
	titles := runVariantTitles()

	for _, want := range []string{"RunContainer", "RunScript", "RunShell", "RunWorkflow"} {
		assert.True(t, slices.Contains(titles, want), "runTask should have variant %q", want)
	}
}

// TestSchemaDefinitionInlineOnly verifies that schemaDefinition supports only
// the inline document form and not the external resource form.
func TestSchemaDefinitionInlineOnly(t *testing.T) {
	s := schemaDefinition

	assert.Len(t, s.OneOf, 1, "schema must have exactly one OneOf branch (inline only)")

	branch := s.OneOf[0]
	assert.Contains(t, branch.Required, "document", "inline schema branch must require 'document'")

	// The external branch would require "resource"; verify it is absent.
	for _, b := range s.OneOf {
		assert.NotContains(t, b.Required, "resource", "schema must not have an external resource branch")
	}
}

// TestRuntimeExpressionPattern verifies that runtimeExpressionDefinition uses
// the expected pattern constant.
func TestRuntimeExpressionPattern(t *testing.T) {
	assert.Equal(t, runtimeExpressionString, runtimeExpressionDefinition.Pattern)
}
