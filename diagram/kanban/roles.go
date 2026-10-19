package kanban

import "github.com/firede/limu/scene"

// The roles of the items of a kanban board.
const (
	RoleAssigned scene.Role = "kanban.assigned"
	RoleCard     scene.Role = "kanban.card"
	RoleColumn   scene.Role = "kanban.column"
	RoleCount    scene.Role = "kanban.count"
	RolePriority scene.Role = "kanban.priority"
	RoleTicket   scene.Role = "kanban.ticket"
	RoleTitle    scene.Role = "kanban.title"
)
