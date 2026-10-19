package class

import "github.com/firede/limu/scene"

// The roles of the items of a class diagram.
const (
	RoleAnnotation     scene.Role = "class.annotation"
	RoleAttribute      scene.Role = "class.attribute"
	RoleCardinality    scene.Role = "class.cardinality"
	RoleClass          scene.Role = "class.class"
	RoleDivider        scene.Role = "class.divider"
	RoleEdgeLabel      scene.Role = "class.edge-label"
	RoleInterface      scene.Role = "class.interface"
	RoleMethod         scene.Role = "class.method"
	RoleNamespace      scene.Role = "class.namespace"
	RoleNamespaceTitle scene.Role = "class.namespace-title"
	RoleNote           scene.Role = "class.note"
	RoleNoteEdge       scene.Role = "class.note-edge"
	RoleRelation       scene.Role = "class.relation"
	RoleTitle          scene.Role = "class.title"
)
