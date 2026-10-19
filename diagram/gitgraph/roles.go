package gitgraph

import "github.com/firede/limu/scene"

// The roles of the items of a git graph.
const (
	RoleBranch      scene.Role = "gitgraph.branch"
	RoleCommit      scene.Role = "gitgraph.commit"
	RoleCommitLabel scene.Role = "gitgraph.commit-label"
	RoleCommitMark  scene.Role = "gitgraph.commit-mark"
	RoleEdge        scene.Role = "gitgraph.edge"
	RoleLane        scene.Role = "gitgraph.lane"
	RoleMerge       scene.Role = "gitgraph.merge"
	RoleTag         scene.Role = "gitgraph.tag"
	RoleTagHole     scene.Role = "gitgraph.tag-hole"
	RoleTitle       scene.Role = "gitgraph.title"
)
