package requirement

import "github.com/firede/limu/scene"

// The roles of the items of a requirement diagram.
const (
	RoleElement           scene.Role = "requirement.element"
	RoleField             scene.Role = "requirement.field"
	RoleKind              scene.Role = "requirement.kind"
	RoleRelationship      scene.Role = "requirement.relationship"
	RoleRelationshipLabel scene.Role = "requirement.relationship-label"
	RoleRequirement       scene.Role = "requirement.requirement"
	RoleTitle             scene.Role = "requirement.title"
	RoleValue             scene.Role = "requirement.value"
)
