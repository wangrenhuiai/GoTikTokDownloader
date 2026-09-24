package app

// Phase 1 unified event protocol. Full task flow arrives in Phase 3.
const (
	EvtTaskCreated   = "task:created"
	EvtTaskUpdated   = "task:updated"
	EvtTaskProgress  = "task:progress"
	EvtTaskCompleted = "task:completed"
	EvtTaskError     = "task:error"
	EvtTaskCancelled = "task:cancelled"
	EvtSystemLog     = "system:log"
	EvtSystemStatus  = "system:status"
)
