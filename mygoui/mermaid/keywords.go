package mermaid

import (
	"strings"
	"sync"

	"github.com/firede/limu/diagram"
)

// Words that start a statement of each type, as Pele's playground lists them for
// completion (website/src/lib/playground-completions.ts, keywords).
const (
	styling = "classDef class style"
	flow    = "subgraph end direction click " + styling + " linkStyle"
)

var starts = map[diagram.Type]string{
	diagram.Flowchart: flow,
	diagram.Swimlane:  flow,
	diagram.AgentFlow: "flow end direction click " + styling + " linkStyle",
	diagram.Sequence: "participant actor Note note activate deactivate loop alt else opt par and critical " +
		"option break rect end box autonumber create destroy link links title",
	diagram.Class: "class namespace direction note click link callback classDef cssClass style",
	diagram.State: "state note direction classDef class",
	diagram.ER:    "direction subgraph end " + styling,
	diagram.Gantt: "title dateFormat axisFormat tickInterval section excludes includes weekday weekend " +
		"todayMarker inclusiveEndDates topAxis click",
	diagram.GitGraph:      "commit branch checkout switch merge cherry-pick",
	diagram.Pie:           "title",
	diagram.Timeline:      "title section",
	diagram.Journey:       "title section",
	diagram.QuadrantChart: "title x-axis y-axis quadrant-1 quadrant-2 quadrant-3 quadrant-4 classDef",
	diagram.XYChart:       "title x-axis y-axis bar line",
	diagram.Requirement: "requirement functionalRequirement interfaceRequirement performanceRequirement " +
		"physicalRequirement designConstraint element direction " + styling,
	diagram.Architecture: "group service junction",
	diagram.C4: "title Person Person_Ext System System_Ext SystemDb SystemDb_Ext SystemQueue SystemQueue_Ext " +
		"Container Container_Ext ContainerDb ContainerDb_Ext ContainerQueue ContainerQueue_Ext Component " +
		"Component_Ext ComponentDb ComponentQueue Boundary Enterprise_Boundary System_Boundary " +
		"Container_Boundary Deployment_Node Node Node_L Node_R Rel BiRel Rel_Back Rel_U Rel_D Rel_L Rel_R " +
		"RelIndex UpdateElementStyle UpdateRelStyle UpdateLayoutConfig",
	diagram.Block:         "columns block end space " + styling,
	diagram.Packet:        "title",
	diagram.Radar:         "title axis curve max min graticule ticks showLegend",
	diagram.Treemap:       "classDef",
	diagram.Cynefin:       "title complex complicated clear chaotic confusion",
	diagram.Wardley:       "title component anchor evolve evolution pipeline note annotation annotations accelerator deaccelerator size",
	diagram.EventModeling: "timeframe tf resetframe rf data",
	diagram.UseCase:       "actor systemBoundary end direction note " + styling,
	diagram.Venn:          "title set union text style",
	diagram.Railroad:      "title",
}

// inline are words that are keywords wherever they are in a statement of the type, not
// only at its start.
var inline = map[diagram.Type]string{
	diagram.Sequence: "as left right of over",
	diagram.State:    "as left right of",
	diagram.Class:    "left right of",
}

// common start a statement of every type.
const common = "accTitle accDescr"

type keywords struct {
	starts, inline map[string]bool
}

var tables = sync.OnceValue(func() map[diagram.Type]keywords {
	set := func(words string) map[string]bool {
		m := map[string]bool{}
		for w := range strings.FieldsSeq(words) {
			m[w] = true
		}
		return m
	}
	t := map[diagram.Type]keywords{}
	var all strings.Builder
	for typ, words := range starts {
		t[typ] = keywords{set(words + " " + common), set(inline[typ])}
		all.WriteString(words + " ")
	}
	// Text whose type is not known yet, or not read, gets every type's words.
	t[""] = keywords{set(all.String() + common), nil}
	return t
})

// keywordsOf is the keywords of a type, or of every type for "".
func keywordsOf(t diagram.Type) keywords {
	if k, ok := tables()[t]; ok {
		return k
	}
	return tables()[""]
}

// directions are the ways a diagram runs, highlighted as constants.
var directions = map[string]bool{"TB": true, "TD": true, "BT": true, "RL": true, "LR": true}
