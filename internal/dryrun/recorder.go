package dryrun

import (
	"fmt"
	"sync"
)

type Recorder interface {
	Record(action, message string)
	Recordf(action, messageFmt string, args ...interface{})
}

type PlannedAction struct {
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}

type SimpleRecorder struct {
	mu             sync.RWMutex // protects fields below
	plannedActions []PlannedAction
}

func (r *SimpleRecorder) Record(action, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plannedActions = append(r.plannedActions, PlannedAction{action, message})
}

func (r *SimpleRecorder) Recordf(action, messageFmt string, args ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plannedActions = append(r.plannedActions, PlannedAction{action, fmt.Sprintf(messageFmt, args...)})
}

func (r *SimpleRecorder) PlannedActions() []PlannedAction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PlannedAction, 0, len(r.plannedActions))
	for _, plannedAction := range r.plannedActions {
		result = append(result, plannedAction)
	}
	return result
}
