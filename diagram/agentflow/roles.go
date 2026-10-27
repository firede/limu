package agentflow

import "github.com/firede/limu/scene"

// The roles of the items of an agent flow.
const (
	RoleCasing        scene.Role = "agentflow.casing" // under an edge that crosses a fill its author sets
	RoleCollapsed     scene.Role = "agentflow.collapsed"
	RoleCollapsedDots scene.Role = "agentflow.collapsed-dots"
	RoleCollapsedRule scene.Role = "agentflow.collapsed-rule"
	RoleEdgeLabel     scene.Role = "agentflow.edge-label"
	RoleFlow          scene.Role = "agentflow.flow"
	RoleTitle         scene.Role = "agentflow.title"
)
