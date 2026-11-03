package limu

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/svg"
)

// Diagrams that set colors in their text, each with the same diagram without the lines that
// set them, and the classes their authors named, which the svg package writes as the classes
// of elements.
var authorColors = []struct {
	name, src, bare string
	classes         []string
}{{
	// classDef fills, a style line on a group that edges run inside and across, which lays
	// casings under them, and a style line on a node.
	name: "flowchart",
	src: `flowchart LR
  subgraph S[Group]
    A --> B
  end
  B --> C
  D --> A
  classDef hot fill:#c00,stroke:#600,stroke-width:3px,color:#fff
  class C hot
  style S fill:#123456,stroke:#f80,stroke-dasharray:4 2
  style D fill:#ff0,stroke-linecap:round`,
	bare: `flowchart LR
  subgraph S[Group]
    A --> B
  end
  B --> C
  D --> A`,
	classes: []string{"hot"},
}, {
	name: "state",
	src: `stateDiagram-v2
  [*] --> Moving
  state Moving {
    Slow --> Fast
  }
  Fast --> Still
  classDef bad fill:#300,color:#fcc
  class Still bad
  style Moving fill:#222`,
	bare: `stateDiagram-v2
  [*] --> Moving
  state Moving {
    Slow --> Fast
  }
  Fast --> Still`,
	classes: []string{"bad"},
}, {
	name: "er",
	src: `erDiagram
  subgraph G
    A ||--o{ B : has
  end
  B ||--|| C : is
  C {
    int id
    string name
  }
  style G fill:#004
  style C fill:#fe0,stroke:#000`,
	bare: `erDiagram
  subgraph G
    A ||--o{ B : has
  end
  B ||--|| C : is
  C {
    int id
    string name
  }`,
}, {
	name: "block",
	src: `block-beta
  columns 3
  block:G
    a b
  end
  c
  a --> c
  style G fill:#333
  style c fill:#9cf`,
	bare: `block-beta
  columns 3
  block:G
    a b
  end
  c
  a --> c`,
}, {
	name: "agentflow",
	src: `agentflow-beta TB
  flow reviewer["Review Agent"]
    changes["Gather changes"]@{ shape: input }
    ok["Clean?"]@{ shape: decision }
    changes --> ok
  end
  done["Done"]
  ok --> done
  style reviewer fill:#003
  style done fill:#fd0`,
	bare: `agentflow-beta TB
  flow reviewer["Review Agent"]
    changes["Gather changes"]@{ shape: input }
    ok["Clean?"]@{ shape: decision }
    changes --> ok
  end
  done["Done"]
  ok --> done`,
}, {
	name: "usecase",
	src: `usecase-beta
  direction LR
  classDef system fill:#050
  classDef order fill:#fcf
  actor Customer("Customer")
  systemBoundary Shop["Order system"]:::system
    Checkout("Place order"):::order
  end
  Customer --> Checkout`,
	bare: `usecase-beta
  direction LR
  actor Customer("Customer")
  systemBoundary Shop["Order system"]
    Checkout("Place order")
  end
  Customer --> Checkout`,
	classes: []string{"system", "order"},
}, {
	name: "sequence",
	src: `sequenceDiagram
  box rgb(20, 40, 80) Team
  participant A
  participant B
  end
  A->>B: Hello
  rect rgb(0, 0, 0)
  B->>A: Hi
  end`,
	bare: `sequenceDiagram
  box Team
  participant A
  participant B
  end
  A->>B: Hello
  rect
  B->>A: Hi
  end`,
}, {
	name: "pie",
	src: `---
config:
  themeVariables:
    pie1: "#101010"
    pie2: transparent
    pie3: currentColor
---
pie title Pets
  "Dogs": 386
  "Cats": 85
  "Rats": 15`,
	bare: `pie title Pets
  "Dogs": 386
  "Cats": 85
  "Rats": 15`,
}, {
	name: "treemap",
	src: `---
config:
  themeVariables:
    cScale0: "#202020"
    cScale1: "#f0e000"
---
treemap-beta
"Section":::frame
  "a": 10:::dark
  "b": 20
"Other"
  "c": 30
classDef frame fill:#FFF9C4,stroke-width:2px
classDef dark fill:navy`,
	bare: `treemap-beta
"Section"
  "a": 10
  "b": 20
"Other"
  "c": 30`,
	classes: []string{"frame", "dark"},
}, {
	name: "mindmap",
	src: `---
config:
  themeVariables:
    cScale1: "#400000"
    cScale2: "#00ff00"
---
mindmap
  root((Root))
    A
      A1
    B`,
	bare: `mindmap
  root((Root))
    A
      A1
    B`,
}, {
	name: "gitGraph",
	src: `---
config:
  themeVariables:
    git0: "#000080"
    git1: "#ffff00"
---
gitGraph
  commit
  branch develop
  commit
  checkout main
  merge develop`,
	bare: `gitGraph
  commit
  branch develop
  commit
  checkout main
  merge develop`,
}, {
	name: "timeline",
	src: `---
config:
  themeVariables:
    cScale0: "#300030"
    cScale1: "#e0ffe0"
---
timeline
  title History
  section Early
    2002 : LinkedIn
  section Later
    2004 : Facebook : Google`,
	bare: `timeline
  title History
  section Early
    2002 : LinkedIn
  section Later
    2004 : Facebook : Google`,
}, {
	name: "radar",
	src: `---
config:
  themeVariables:
    cScale0: "#880000"
    cScale1: "rgba(0, 0, 255, 0.5)"
---
radar-beta
  axis A, B, C
  curve c1{1, 2, 3}
  curve c2{3, 2, 1}`,
	bare: `radar-beta
  axis A, B, C
  curve c1{1, 2, 3}
  curve c2{3, 2, 1}`,
}, {
	name: "venn",
	src: `---
config:
  themeVariables:
    venn1: "#aa0000"
    venn2: "#0000aa"
---
venn-beta
  set A
  set B
  union A,B`,
	bare: `venn-beta
  set A
  set B
  union A,B`,
}, {
	name: "xychart",
	src: `---
config:
  themeVariables:
    xyChart:
      plotColorPalette: "#ff0000, #00aa00"
---
xychart-beta
  x-axis [jan, feb, mar]
  bar [50, 60, 70]
  line [40, 80, 30]`,
	bare: `xychart-beta
  x-axis [jan, feb, mar]
  bar [50, 60, 70]
  line [40, 80, 30]`,
}}

// classAttr is a class attribute the svg package writes.
var classAttr = regexp.MustCompile(` class="([^"]*)"`)

// withoutClasses is an SVG document with the classes of elements named left out of their
// class attributes.
func withoutClasses(doc string, names []string) string {
	return classAttr.ReplaceAllStringFunc(doc, func(attr string) string {
		var kept []string
		for _, c := range strings.Fields(classAttr.FindStringSubmatch(attr)[1]) {
			if !slices.Contains(names, c) {
				kept = append(kept, c)
			}
		}
		if len(kept) == 0 {
			return ""
		}
		return ` class="` + strings.Join(kept, " ") + `"`
	})
}

// A look that is ThemeOnly draws a diagram that sets colors in its text as it draws the same
// diagram without them: what the author's style lines, classes and palette set, the colors
// picked for what lies on the author's fills, and the casings, are all left out. A look that
// isn't draws the colors.
func TestThemeOnly(t *testing.T) {
	only := &svg.Options{Look: style.Look{ThemeOnly: true}}
	for _, c := range authorColors {
		sc, err := Layout(c.src, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		bare, err := Layout(c.bare, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := withoutClasses(svg.Render(sc, only), c.classes)
		if want := svg.Render(bare, nil); got != want {
			t.Errorf("%s drawn ThemeOnly:\n%s\nwithout its colors:\n%s", c.name, got, want)
		}
		if drawn := withoutClasses(svg.Render(sc, nil), c.classes); drawn == got {
			t.Errorf("%s draws no color of its author's", c.name)
		}
	}
}

// The flowchart above lays casings under the edges that cross its group's fill, which a
// ThemeOnly look doesn't draw.
func TestThemeOnlyCasings(t *testing.T) {
	sc, err := Layout(authorColors[0].src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(sc.Items, func(it scene.Item) bool { return strings.HasSuffix(string(it.Role), ".casing") }) {
		t.Fatal("no casing laid out")
	}
	look := style.Look{ThemeOnly: true}
	if look.Draws("flowchart.casing") || !look.Draws("flowchart.edge") {
		t.Error("ThemeOnly draws a casing or leaves out an edge")
	}
	if !(&style.Look{}).Draws("flowchart.casing") {
		t.Error("a look that isn't ThemeOnly leaves out a casing")
	}
}
