package gantt

import "github.com/firede/limu/scene"

// The roles of the items of a gantt chart.
const (
	RoleAxis         scene.Role = "gantt.axis"
	RoleExcluded     scene.Role = "gantt.excluded"
	RoleGrid         scene.Role = "gantt.grid"
	RoleSection      scene.Role = "gantt.section"
	RoleSectionLabel scene.Role = "gantt.section-label"
	RoleTask         scene.Role = "gantt.task"
	RoleTick         scene.Role = "gantt.tick"
	RoleTitle        scene.Role = "gantt.title"
	RoleToday        scene.Role = "gantt.today"
	RoleVert         scene.Role = "gantt.vert"
	RoleVertLabel    scene.Role = "gantt.vert-label"
)
