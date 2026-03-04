package agentgraph

import (
	"fmt"
	"strings"

	"github.com/c360studio/semstreams/pkg/types"
)

// Entity ID component constants. These define the fixed hierarchy positions
// used when constructing 6-part graph entity IDs for agentic resources.
//
// Format: org.platform.domain.system.type.instance
// Example loop:  semsage.default.agentic.orchestrator.loop.<loopID>
// Example task:  semsage.default.agentic.orchestrator.task.<taskID>
const (
	OrgDefault         = "semsage"
	PlatformDefault    = "default"
	DomainAgentic      = "agentic"
	SystemOrchestrator = "orchestrator"
	TypeLoop           = "loop"
	TypeTask           = "task"

	// Relationship predicates follow the three-level dotted format: domain.category.property.

	// PredicateSpawned records a parent loop spawning a child loop.
	// Direction: parent loop entity -> child loop entity.
	PredicateSpawned = "agentic.loop.spawned"

	// PredicateLoopTask records the association between a loop and a task it owns.
	// Direction: loop entity -> task entity.
	PredicateLoopTask = "agentic.loop.task"

	// PredicateDependsOn records a task-to-task dependency (DAG edge).
	// Direction: dependent task entity -> prerequisite task entity.
	PredicateDependsOn = "agentic.task.depends_on"

	// Entity property predicates describe scalar attributes of loop entities.

	// PredicateRole records the functional role of a loop (e.g., "planner", "executor").
	PredicateRole = "agentic.loop.role"

	// PredicateModel records the LLM model identifier used by a loop.
	PredicateModel = "agentic.loop.model"

	// PredicateStatus records the current lifecycle status of a loop.
	PredicateStatus = "agentic.loop.status"

	// SourceSemsage is the source identifier stamped on triples created by Semsage.
	// It enables provenance filtering when querying the graph.
	SourceSemsage = "semsage"
)

// ValidateInstanceID checks that an instance ID is valid for use in a 6-part entity ID.
// It must be non-empty and must not contain dots (which would break the 6-part format).
func ValidateInstanceID(id string) error {
	if id == "" {
		return fmt.Errorf("agentgraph: instance ID must not be empty")
	}
	if strings.Contains(id, ".") {
		return fmt.Errorf("agentgraph: instance ID %q must not contain dots", id)
	}
	return nil
}

// LoopEntityID returns the full 6-part graph entity ID string for an agent loop.
// Format: semsage.default.agentic.orchestrator.loop.<loopID>
// Panics if loopID is empty or contains dots.
func LoopEntityID(loopID string) string {
	if err := ValidateInstanceID(loopID); err != nil {
		panic(err)
	}
	return LoopEntityIDParsed(loopID).String()
}

// TaskEntityID returns the full 6-part graph entity ID string for a task.
// Format: semsage.default.agentic.orchestrator.task.<taskID>
// Panics if taskID is empty or contains dots.
func TaskEntityID(taskID string) string {
	if err := ValidateInstanceID(taskID); err != nil {
		panic(err)
	}
	return types.EntityID{
		Org:      OrgDefault,
		Platform: PlatformDefault,
		Domain:   DomainAgentic,
		System:   SystemOrchestrator,
		Type:     TypeTask,
		Instance: taskID,
	}.String()
}

// LoopEntityIDParsed returns a structured EntityID for an agent loop.
// Prefer LoopEntityID when only the string form is needed.
func LoopEntityIDParsed(loopID string) types.EntityID {
	return types.EntityID{
		Org:      OrgDefault,
		Platform: PlatformDefault,
		Domain:   DomainAgentic,
		System:   SystemOrchestrator,
		Type:     TypeLoop,
		Instance: loopID,
	}
}

// LoopTypePrefix returns the 5-part prefix that identifies the loop entity type.
// Use this prefix with EntityManager.ListWithPrefix to enumerate all loop entities.
// Format: semsage.default.agentic.orchestrator.loop
func LoopTypePrefix() string {
	// TypePrefix() returns "org.platform.domain.system.type" — no instance component.
	return LoopEntityIDParsed("_").TypePrefix()
}

// TaskTypePrefix returns the 5-part prefix that identifies the task entity type.
// Use this prefix with EntityManager.ListWithPrefix to enumerate all task entities.
// Format: semsage.default.agentic.orchestrator.task
func TaskTypePrefix() string {
	eid := types.EntityID{
		Org:      OrgDefault,
		Platform: PlatformDefault,
		Domain:   DomainAgentic,
		System:   SystemOrchestrator,
		Type:     TypeTask,
		Instance: "_",
	}
	// TypePrefix() returns "org.platform.domain.system.type" — no instance component.
	return eid.TypePrefix()
}
