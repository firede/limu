package limu

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// A diagram of each type limu reads, with something of each to lay out.
var samples = map[diagram.Type]string{
	diagram.Flowchart:     "flowchart LR\n  A[Text] --> B(Scene) --> C{Backend}\n  subgraph S\n    D --> E\n  end",
	diagram.Pie:           "pie title Pets\n  \"Dogs\": 386\n  \"Cats\": 85\n  \"Rats\": 15",
	diagram.ER:            "erDiagram\n  CUSTOMER ||--o{ ORDER : places\n  ORDER {\n    string id PK\n  }",
	diagram.Mindmap:       "mindmap\n  root((limu))\n    Text\n      Labels\n    Scene",
	diagram.Kanban:        "kanban\n  Todo\n    [Parse]\n    [Lay out]\n  Done\n    [Draw]",
	diagram.Timeline:      "timeline\n  title History\n  2002 : LinkedIn\n  2004 : Facebook : Google",
	diagram.Journey:       "journey\n  title A day\n  section Work\n    Make tea: 5: Me\n    Write: 3: Me, Cat",
	diagram.Class:         "classDiagram\n  Animal <|-- Duck\n  class Duck {\n    +String beak\n    +swim()\n  }",
	diagram.QuadrantChart: "quadrantChart\n  title Reach\n  x-axis Low --> High\n  y-axis Low --> High\n  A: [0.3, 0.6]\n  B: [0.45, 0.23]",
	diagram.XYChart:       "xychart-beta\n  title Sales\n  x-axis [jan, feb, mar]\n  y-axis 0 --> 100\n  bar [50, 60, 70]\n  line [40, 80, 30]",
	diagram.State:         "stateDiagram-v2\n  [*] --> Still\n  Still --> Moving\n  state Moving {\n    Slow --> Fast\n  }\n  Moving --> [*]",
	diagram.Gantt:         "gantt\n  dateFormat YYYY-MM-DD\n  section A\n  Task one :a1, 2024-01-01, 3d\n  Task two :after a1, 2d",
	diagram.Architecture:  "architecture-beta\n  group api(cloud)[API]\n  service db(database)[Database] in api\n  service server(server)[Server] in api\n  db:L -- R:server",
	diagram.Requirement:   "requirementDiagram\n  requirement test_req {\n    id: 1\n    text: the test text.\n    risk: high\n    verifymethod: test\n  }\n  element test_entity {\n    type: simulation\n  }\n  test_entity - satisfies -> test_req",
	diagram.Sankey:        "sankey-beta\n  a,b,5\n  a,c,3\n  b,d,2",
	diagram.Packet:        "packet-beta\n  0-15: \"Source Port\"\n  16-31: \"Destination Port\"",
	diagram.Radar:         "radar-beta\n  axis A, B, C\n  curve c1{1, 2, 3}\n  curve c2{3, 2, 1}",
	diagram.Treemap:       "treemap-beta\n  \"Section\"\n    \"Leaf A\": 10\n    \"Leaf B\": 20",
	diagram.Info:          "info",
	diagram.Sequence:      "sequenceDiagram\n  Alice->>Bob: Hello\n  loop Every minute\n    Bob-->>Alice: Hi\n  end\n  Note over Alice,Bob: A note",
	diagram.GitGraph:      "gitGraph\n  commit\n  branch develop\n  commit\n  checkout main\n  merge develop",
	diagram.C4:            "C4Context\n  title System\n  Person(a, \"Customer\")\n  System(b, \"Bank\")\n  Rel(a, b, \"Uses\")",
	diagram.Block:         "block-beta\n  columns 3\n  a b c\n  a --> b",
	diagram.Railroad:      "railroad-ebnf-beta\n  digit = \"0\" | \"1\" , [ \"2\" ] ;",
	diagram.AgentFlow:     "agentflow-beta TB\n  flow reviewer[\"Review Agent\"]\n    changes[\"Gather changes\"]@{ shape: input }\n    ok[\"Clean?\"]@{ shape: decision }\n    changes --> ok\n  end",
	diagram.TreeView:      "treeView-beta\n  root\n    src\n      main.go\n    README.md",
	diagram.Cynefin:       "cynefin-beta\n  clear\n    \"Follow the steps\"\n  complex\n    \"Probe\"",
	diagram.Wardley:       "wardley-beta\n  title Tea\n  component Cup [0.73, 0.78]\n  component Tea [0.63, 0.81]\n  Cup -> Tea",
	diagram.EventModeling: "eventmodeling\n  tf 01 ui Form\n  tf 02 cmd Submit\n  tf 03 evt Submitted",
	diagram.UseCase:       "usecase-beta\n  direction LR\n  actor Customer(\"Customer\")\n  systemBoundary \"Order system\"\n    Checkout(\"Place order\")\n  end\n  Customer --> Checkout",
	diagram.Venn:          "venn-beta\n  set A\n  set B\n  union A,B",
	diagram.Ishikawa:      "ishikawa-beta\n  Late delivery\n    People\n      Too few drivers\n    Machines\n      Old trucks",
	diagram.Swimlane:      "swimlane-beta LR\n  subgraph a [Lane A]\n    x[Start]\n  end\n  subgraph b [Lane B]\n    y[End]\n  end\n  x --> y",
}

func parsed(t *testing.T, typ diagram.Type) *Diagram {
	t.Helper()
	src, ok := samples[typ]
	if !ok {
		t.Fatalf("no sample of %s", typ)
	}
	d, err := Parse(src, diagram.Options{})
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	if d.Type() != typ {
		t.Fatalf("Parse(%q) is a %s, want a %s", src, d.Type(), typ)
	}
	return d
}

func TestTypes(t *testing.T) {
	if len(types) != len(kinds) || len(types) != 33 {
		t.Fatalf("%d types and %d kinds, want 33 of each", len(types), len(kinds))
	}
	for _, typ := range Types() {
		if _, ok := kinds[typ]; !ok {
			t.Errorf("%s is listed but has no kind", typ)
		}
		if !Supports(samples[typ]) {
			t.Errorf("Supports(%q) is false", samples[typ])
		}
	}
	got := Types()
	got[0] = "changed"
	if Types()[0] != diagram.Flowchart {
		t.Error("Types gives its own slice")
	}
	for _, src := range []string{"", "not a diagram", "---\na: [\n---\nflowchart\n  A"} {
		if Supports(src) {
			t.Errorf("Supports(%q) is true", src)
		}
	}
}

// Every diagram package's Layout takes nil options as the zero Options, as limu's own does.
func TestLayoutWithNilOptions(t *testing.T) {
	for _, typ := range types {
		d := parsed(t, typ)
		sc, err := d.kind.layout(d.model, d.config, nil)
		if err != nil {
			t.Errorf("%s: %v", typ, err)
			continue
		}
		want, err := d.Layout(&layout.Options{})
		if err != nil {
			t.Fatalf("%s: %v", typ, err)
		}
		if !reflect.DeepEqual(sc, want) {
			t.Errorf("%s: laid out with nil options unlike with the zero Options", typ)
		}
	}
}

// One Diagram laid out many times, with other options and at once, gives what a Diagram
// parsed anew for each gives.
func TestLayoutManyTimes(t *testing.T) {
	// A gantt chart draws today where the present is.
	now := time.Date(2024, 1, 3, 12, 0, 0, 0, time.UTC)
	opts := []*layout.Options{
		{Now: now},
		{Now: now, MaxWidth: 300},
		{Now: now, FontSize: 20, Snap: 1},
		{Now: now, MaxWidth: 200, KeepDirection: true, Padding: new(0.0)},
		{Now: now, Padding: new(20.0), Routing: layout.Orthogonal},
	}
	for _, typ := range types {
		d := parsed(t, typ)
		want := make([]any, len(opts))
		for i, o := range opts {
			fresh := parsed(t, typ)
			want[i], _ = fresh.Layout(o)
		}
		var wg sync.WaitGroup
		got := make([]any, 4*len(opts))
		for i := range got {
			wg.Go(func() {
				sc, err := d.Layout(opts[i%len(opts)])
				if err != nil {
					t.Errorf("%s: %v", typ, err)
				}
				got[i] = sc
			})
		}
		wg.Wait()
		for i := range got {
			if !reflect.DeepEqual(got[i], want[i%len(opts)]) {
				t.Errorf("%s: laid out again with options %d unlike a diagram parsed anew", typ, i%len(opts))
			}
		}
	}
}

func TestParseOptions(t *testing.T) {
	// The host's configuration is under the diagram's own.
	src := "flowchart LR\n  A --> B"
	d, err := Parse(src, diagram.Options{Config: diagram.Config{"flowchart": diagram.Config{"curve": "linear"}, "seed": 3.0}})
	if err != nil {
		t.Fatal(err)
	}
	if fc, _ := d.Config()["flowchart"].(diagram.Config); fc["curve"] != "linear" || d.Config()["seed"] != 3.0 {
		t.Errorf("Config() = %v", d.Config())
	}
	_, err = Parse(src, diagram.Options{Limit: 5})
	var e *diagram.Error
	if !errors.As(err, &e) || e.Code != diagram.Limit {
		t.Errorf("Parse past a limit of 5: %v", err)
	}
	if _, err := Parse(src, diagram.Options{Limit: -1}); err != nil {
		t.Errorf("Parse with no limit: %v", err)
	}
}

// A scene with a number that is not finite is an error, not a scene.
func TestLayoutNotFinite(t *testing.T) {
	sc, err := Layout("sankey-beta\na,b,5e-324", nil)
	var e *diagram.Error
	if !errors.As(err, &e) || e.Code != diagram.Limit || e.Type != diagram.Sankey || sc != nil {
		t.Errorf("Layout of a flow of 5e-324 = %v, %v", sc, err)
	}
}

// Bytes that are not UTF-8 are read as a browser's decoder reads them.
func TestLayoutInvalidUTF8(t *testing.T) {
	got, err := Layout("flowchart\n  A[a\xffb]", nil)
	if err != nil {
		t.Fatal(err)
	}
	want, err := Layout("flowchart\n  A[a�b]", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("A[a\\xffb] is laid out unlike A[a\\uFFFDb]")
	}
	if i := slices.IndexFunc(got.Items, func(it scene.Item) bool { return it.Label != nil }); i < 0 || got.Items[i].Label.Lines[0][0].Text != "a�b" {
		t.Errorf("the node's label is not a\\uFFFDb")
	}
}

// Snap in the options is applied to the scene.
func TestLayoutSnaps(t *testing.T) {
	d := parsed(t, diagram.Flowchart)
	loose, _ := d.Layout(nil)
	snapped, _ := d.Layout(&layout.Options{Snap: 4})
	loose.Snap(4)
	if !reflect.DeepEqual(loose, snapped) {
		t.Error("Layout with Snap 4 is not the scene snapped to 4")
	}
}

// Padding moves what a diagram draws and widens it, as Pele's padding option does; nil is
// Pele's 8. The render suites of flowcharts, pie charts and treemaps check paddings of 0, 2.5
// and 20 against Pele, number for number.
func TestLayoutPadding(t *testing.T) {
	d := parsed(t, diagram.Pie)
	def, _ := d.Layout(nil)
	eight, _ := d.Layout(&layout.Options{Padding: new(8.0)})
	if !reflect.DeepEqual(def, eight) {
		t.Error("nil padding is not a padding of 8")
	}
	none, _ := d.Layout(&layout.Options{Padding: new(0.0)})
	wide, _ := d.Layout(&layout.Options{Padding: new(20.0)})
	if wide.Width-none.Width != 40 || wide.Height-none.Height != 40 || none.Width != def.Width-16 {
		t.Errorf("sizes %vx%v at 0, %vx%v at 8, %vx%v at 20", none.Width, none.Height, def.Width, def.Height, wide.Width, wide.Height)
	}
	// Pele adds the padding to each place, so the last bit may differ.
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	for i := range none.Items {
		a, b := none.Items[i], wide.Items[i]
		moved := near(b.At.X-a.At.X, 20) && near(b.At.Y-a.At.Y, 20)
		if a.Box != (scene.Rect{}) {
			moved = moved && near(b.Box.X-a.Box.X, 20) && near(b.Box.Y-a.Box.Y, 20) && near(a.Box.W, b.Box.W) && near(a.Box.H, b.Box.H)
		}
		if !moved {
			t.Errorf("item %d at %v, %v with no padding, at %v, %v with 20", i, a.Box, a.At, b.Box, b.At)
		}
	}
}
