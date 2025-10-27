package customcmd

// ManagerAdapter adapts Manager to work with the agent interface
// This avoids circular dependencies between agent and customcmd packages
type ManagerAdapter struct {
	manager *Manager
}

// NewManagerAdapter creates a new adapter
func NewManagerAdapter(manager *Manager) *ManagerAdapter {
	return &ManagerAdapter{manager: manager}
}

// GetRelevantDocs returns relevant docs for the agent (interface method)
func (a *ManagerAdapter) GetRelevantDocs(request string, maxDocs int) []interface{} {
	if a.manager == nil {
		return nil
	}

	agentDocs := a.manager.GetRelevantDocsForAgent(request, maxDocs)

	// Convert to interface{} slice
	result := make([]interface{}, len(agentDocs))
	for i := range agentDocs {
		result[i] = agentDocs[i]
	}

	return result
}

// IsIndexed returns whether commands have been indexed
func (a *ManagerAdapter) IsIndexed() bool {
	if a.manager == nil {
		return false
	}
	return a.manager.IsIndexed()
}
