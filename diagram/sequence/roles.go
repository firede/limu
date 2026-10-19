package sequence

import "github.com/firede/limu/scene"

// The roles of the items of a sequence diagram.
const (
	RoleActivation   scene.Role = "sequence.activation"
	RoleActor        scene.Role = "sequence.actor"
	RoleBox          scene.Role = "sequence.box"
	RoleCondition    scene.Role = "sequence.condition"
	RoleDestroy      scene.Role = "sequence.destroy"
	RoleDivider      scene.Role = "sequence.divider"
	RoleFrame        scene.Role = "sequence.frame"
	RoleFrameKind    scene.Role = "sequence.frame-kind"
	RoleLifeline     scene.Role = "sequence.lifeline"
	RoleMessage      scene.Role = "sequence.message"
	RoleMessageLabel scene.Role = "sequence.message-label"
	RoleNote         scene.Role = "sequence.note"
	RoleNumber       scene.Role = "sequence.number"
	RoleRect         scene.Role = "sequence.rect"
	RoleTitle        scene.Role = "sequence.title"
)
