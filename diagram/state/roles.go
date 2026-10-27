package state

import "github.com/firede/limu/scene"

// The roles of the items of a state diagram.
const (
	RoleCasing      scene.Role = "state.casing" // under a transition that crosses a fill its author sets
	RoleChoice      scene.Role = "state.choice"
	RoleComposite   scene.Role = "state.composite"
	RoleDescription scene.Role = "state.description"
	RoleDivider     scene.Role = "state.divider"
	RoleEdgeLabel   scene.Role = "state.edge-label"
	RoleEnd         scene.Role = "state.end"
	RoleFork        scene.Role = "state.fork"
	RoleJoin        scene.Role = "state.join"
	RoleNote        scene.Role = "state.note"
	RoleNoteEdge    scene.Role = "state.note-edge"
	RoleStart       scene.Role = "state.start"
	RoleState       scene.Role = "state.state"
	RoleTitle       scene.Role = "state.title"
	RoleTransition  scene.Role = "state.transition"
)
