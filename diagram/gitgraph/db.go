package gitgraph

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// fail panics with Pele's semantic error, which buildGit recovers.
func fail(message string) {
	panic(diagram.NewError(message, diagram.Semantic, "gitGraph"))
}

// typeCode is the commitType of a type's name, as Pele's typeCode gives it.
func typeCode(name string) int {
	switch name {
	case "REVERSE":
		return Reverse
	case "HIGHLIGHT":
		return Highlight
	}
	return Normal
}

// builder replays a git graph's statements, as the database buildGit makes in Pele does with
// the calls Mermaid's gitGraphParser makes on it.
type builder struct {
	commits  []*Commit
	index    map[string]int // of a commit in commits, by id
	branches []Branch
	bindex   map[string]int // of a branch in branches, by name
	orders   []float64      // of the branches, in their order
	warnings []string
	head     *Commit
	current  string
	dir      string
	seq      int
}

func (b *builder) commit(id string) (*Commit, bool) {
	if i, ok := b.index[id]; ok {
		return b.commits[i], true
	}
	return nil, false
}

func (b *builder) branch(name string) (*Branch, bool) {
	if i, ok := b.bindex[name]; ok {
		return &b.branches[i], true
	}
	return nil, false
}

// setBranch sets a branch's head, adding the branch where it is new, as a Map's set does.
func (b *builder) setBranch(name, head string, has bool) {
	if br, ok := b.branch(name); ok {
		br.Head, br.HasHead = head, has
		return
	}
	b.bindex[name] = len(b.branches)
	b.branches = append(b.branches, Branch{Name: name, Head: head, HasHead: has})
}

// nextID is an id for a commit the text gives none, as Pele's nextId makes it. Mermaid
// appends seven random hex digits to the sequence number; Pele derives them from the sequence
// number instead, so that the same text always gives the same ids.
func (b *builder) nextID() string {
	for salt := uint64(b.seq + 1); ; salt += 0x9e3779b1 {
		h := uint32(salt) * 0x9e3779b1
		id := strconv.Itoa(b.seq) + "-" + fmt.Sprintf("%08x", h)[:7]
		if _, ok := b.index[id]; !ok {
			return id
		}
	}
}

// add makes a commit the head of the current branch, in the place of one with its id.
func (b *builder) add(c *Commit) {
	b.head = c
	if i, ok := b.index[c.ID]; ok {
		b.commits[i] = c
	} else {
		b.index[c.ID] = len(b.commits)
		b.commits = append(b.commits, c)
	}
	b.setBranch(b.current, c.ID, true)
}

func (b *builder) headParents(more ...string) []string {
	if b.head == nil {
		return []string{}
	}
	return append([]string{b.head.ID}, more...)
}

// checkout is Pele's checkout.
func (b *builder) checkout(name string) {
	br, ok := b.branch(name)
	if !ok {
		fail(`Trying to checkout branch which is not yet created. (Help try using "branch ` + name + `")`)
	}
	b.current = name
	b.head = nil
	if br.HasHead && br.Head != "" {
		b.head, _ = b.commit(br.Head)
	}
}

// addCommit is the database's commit.
func (b *builder) addCommit(msg string, id *string, typ int, tags []string) {
	c := &Commit{}
	if id != nil && *id != "" {
		c.ID = *id
	} else {
		c.ID = b.nextID()
	}
	c.Message = msg
	c.Seq = b.seq
	b.seq++
	c.Type = typ
	c.Tags = slices.Clone(tags)
	if c.Tags == nil {
		c.Tags = []string{}
	}
	c.Parents = b.headParents()
	c.Branch = b.current
	if _, ok := b.index[c.ID]; ok {
		b.warnings = append(b.warnings, "Commit ID "+c.ID+" already exists")
	}
	b.add(c)
}

// addBranch is the database's branch.
func (b *builder) addBranch(name string, order float64) {
	if _, ok := b.branch(name); ok {
		fail(`Trying to create an existing branch. (Help: Either use a new name if you want create a new branch or try using "checkout ` + name + `")`)
	}
	if b.head != nil {
		b.setBranch(name, b.head.ID, true)
	} else {
		b.setBranch(name, "", false)
	}
	b.orders = append(b.orders, order)
	b.checkout(name)
}

// merge is the database's merge.
func (b *builder) merge(other, customID string, typ *int, tags []string) {
	cur, _ := b.branch(b.current)
	var currentHead, otherHead string
	if cur != nil && cur.HasHead {
		currentHead = cur.Head
	}
	ob, otherExists := b.branch(other)
	if ob != nil && ob.HasHead {
		otherHead = ob.Head
	}
	var currentCommit, otherCommit *Commit
	if currentHead != "" {
		currentCommit, _ = b.commit(currentHead)
	}
	if otherHead != "" {
		otherCommit, _ = b.commit(otherHead)
	}
	if currentCommit != nil && otherCommit != nil && currentCommit.Branch == other {
		fail("Cannot merge branch '" + other + "' into itself.")
	}
	if b.current == other {
		fail(`Incorrect usage of "merge". Cannot merge a branch to itself`)
	}
	if currentCommit == nil {
		fail(`Incorrect usage of "merge". Current branch (` + b.current + `)has no commits`)
	}
	if !otherExists {
		fail(`Incorrect usage of "merge". Branch to be merged (` + other + `) does not exist`)
	}
	if otherCommit == nil {
		fail(`Incorrect usage of "merge". Branch to be merged (` + other + `) has no commits`)
	}
	if currentCommit == otherCommit {
		fail(`Incorrect usage of "merge". Both branches have same head`)
	}
	if customID != "" {
		if _, ok := b.index[customID]; ok {
			fail(`Incorrect usage of "merge". Commit with id:` + customID + ` already exists, use different custom id`)
		}
	}
	c := &Commit{}
	if customID != "" {
		c.ID = customID
	} else {
		c.ID = b.nextID()
	}
	c.Message = "merged branch " + other + " into " + b.current
	c.Seq = b.seq
	b.seq++
	c.Parents = b.headParents(otherHead)
	c.Branch = b.current
	c.Type = MergeType
	c.CustomType = typ
	custom := customID != ""
	c.CustomID = &custom
	c.Tags = slices.Clone(tags)
	if c.Tags == nil {
		c.Tags = []string{}
	}
	b.add(c)
}

// cherryPick is the database's cherryPick. tags is nil where the statement gives none.
func (b *builder) cherryPick(sourceID *string, targetID string, tags []string, parent *string) {
	var source *Commit
	if sourceID != nil && *sourceID != "" {
		source, _ = b.commit(*sourceID)
	}
	if source == nil {
		fail(`Incorrect usage of "cherryPick". Source commit id should exist and provided`)
	}
	hasParent := parent != nil && *parent != ""
	if hasParent && !slices.Contains(source.Parents, *parent) {
		fail("Invalid operation: The specified parent commit is not an immediate parent of the cherry-picked commit.")
	}
	if source.Type == MergeType && !hasParent {
		fail("Incorrect usage of cherry-pick: If the source commit is a merge commit, an immediate parent commit must be specified.")
	}
	if targetID != "" {
		if _, ok := b.index[targetID]; ok {
			return
		}
	}
	if source.Branch == b.current {
		fail(`Incorrect usage of "cherryPick". Source commit is already on current branch`)
	}
	cur, _ := b.branch(b.current)
	if cur == nil || !cur.HasHead || cur.Head == "" {
		fail(`Incorrect usage of "cherry-pick". Current branch (` + b.current + `)has no commits`)
	}
	if _, ok := b.index[cur.Head]; !ok {
		fail(`Incorrect usage of "cherry-pick". Current branch (` + b.current + `)has no commits`)
	}
	c := &Commit{ID: b.nextID()}
	c.Message = "cherry-picked " + source.Message + " into " + b.current
	c.Seq = b.seq
	b.seq++
	c.Parents = b.headParents(source.ID)
	c.Branch = b.current
	c.Type = CherryPick
	if tags != nil {
		c.Tags = []string{}
		for _, t := range tags {
			if t != "" {
				c.Tags = append(c.Tags, t)
			}
		}
	} else {
		tag := "cherry-pick:" + source.ID
		if source.Type == MergeType {
			p := "undefined"
			if parent != nil {
				p = *parent
			}
			tag += "|parent:" + p
		}
		c.Tags = []string{tag}
	}
	b.add(c)
}

// statement replays a statement, as Pele's parseStatement does.
func (b *builder) statement(s *Statement) {
	switch s.Kind {
	case "Commit":
		msg := ""
		if s.Message != nil {
			msg = *s.Message
		}
		typ := Normal
		if s.Type != nil {
			typ = typeCode(*s.Type)
		}
		b.addCommit(msg, s.ID, typ, s.Tags)
	case "Branch":
		order := 0.0
		if s.Order != nil {
			order = *s.Order
		}
		b.addBranch(s.Name, order)
	case "Merge":
		id := ""
		if s.ID != nil {
			id = *s.ID
		}
		var typ *int
		if s.Type != nil {
			t := typeCode(*s.Type)
			typ = &t
		}
		b.merge(s.Branch, id, typ, s.Tags)
	case "Checkout":
		b.checkout(s.Branch)
	case "CherryPicking":
		tags := s.Tags
		if len(tags) == 0 {
			tags = nil
		}
		b.cherryPick(s.ID, "", tags, s.Parent)
	}
}

// buildGit replays a git graph's tree into its model, as Pele's buildGit does: config is the
// diagram's configuration and title the title its front matter gives. An error is a
// *diagram.Error.
func buildGit(ast *AST, config diagram.Config, title string) (m *Model, err error) {
	defer func() {
		if v := recover(); v != nil {
			e, ok := v.(*diagram.Error)
			if !ok {
				panic(v)
			}
			m, err = nil, e
		}
	}()
	git, _ := config["gitGraph"].(diagram.Config)
	mainName := "main"
	if s, ok := git["mainBranchName"].(string); ok {
		mainName = s
	}
	mainOrder := 0.0
	if v, ok := git["mainBranchOrder"].(float64); ok {
		mainOrder = v
	}

	b := &builder{index: map[string]int{}, bindex: map[string]int{}, current: mainName, dir: "LR"}
	b.setBranch(mainName, "", false)
	b.orders = append(b.orders, mainOrder)
	if ast.Dir != "" {
		b.dir = ast.Dir
	}
	for i := range ast.Statements {
		b.statement(&ast.Statements[i])
	}

	// A branch made without an order sorts by creation among those of order zero, as in
	// Mermaid; every branch the statements make has an order, zero where they give none.
	type lane struct {
		name  string
		order float64
	}
	lanes := make([]lane, len(b.branches))
	for i, br := range b.branches {
		lanes[i] = lane{br.Name, b.orders[i]}
	}
	// Pele sorts with a.order - b.order, and V8 takes a NaN it gives as not less; here a NaN
	// compares equal. With NaNs anywhere the two sorts could order lanes differently, but the
	// only order that can be NaN is the main branch's, from mainBranchOrder: .nan, and the main
	// branch is first, where both leave it and sort the rest.
	slices.SortStableFunc(lanes, func(x, y lane) int {
		d := x.order - y.order
		switch {
		case d < 0:
			return -1
		case d > 0:
			return 1
		}
		return 0
	})

	m = &Model{Direction: b.dir, Commits: b.commits, Branches: b.branches, CurrentBranch: b.current,
		Head: b.head, Warnings: b.warnings}
	for _, l := range lanes {
		m.Lanes = append(m.Lanes, l.name)
	}
	if ast.Title != nil && *ast.Title != "" {
		m.Title = *ast.Title
	} else {
		m.Title = title
	}
	if ast.AccTitle != nil {
		m.AccTitle = chars.TrimStart(*ast.AccTitle)
	}
	if ast.AccDescr != nil {
		m.AccDescr = chars.DropIndents(*ast.AccDescr)
	}
	return m, nil
}
