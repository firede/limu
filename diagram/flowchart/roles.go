package flowchart

import "github.com/firede/limu/scene"

// The roles of the items of a flowchart.
const (
	RoleCasing       scene.Role = "flowchart.casing" // under an edge that crosses a fill its author sets
	RoleCluster      scene.Role = "flowchart.cluster"
	RoleClusterTitle scene.Role = "flowchart.cluster-title"
	RoleEdge         scene.Role = "flowchart.edge"
	RoleEdgeLabel    scene.Role = "flowchart.edge-label"
	RoleNode         scene.Role = "flowchart.node"
	RoleTitle        scene.Role = "flowchart.title"
)
