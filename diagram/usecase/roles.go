package usecase

import "github.com/firede/limu/scene"

// The roles of the items of a use case diagram.
const (
	RoleActor scene.Role = "usecase.actor"
	// RoleCasing is a casing under a relation that crosses the border of a fill an author
	// sets, drawn in the background so that the relation stands out on the fill.
	RoleCasing     scene.Role = "usecase.casing"
	RoleEdgeLabel  scene.Role = "usecase.edge-label"
	RoleJson       scene.Role = "usecase.json"
	RoleJsonKey    scene.Role = "usecase.json-key"
	RoleJsonValue  scene.Role = "usecase.json-value"
	RoleNote       scene.Role = "usecase.note"
	RoleNoteLink   scene.Role = "usecase.note-link"
	RoleRelation   scene.Role = "usecase.relation"
	RoleStereotype scene.Role = "usecase.stereotype"
	RoleSystem     scene.Role = "usecase.system"
	RoleTitle      scene.Role = "usecase.title"
	RoleUsecase    scene.Role = "usecase.usecase"
)
