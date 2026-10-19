package common

// Clickable splits the classes Pele gives a node into those its author gave it and its
// tags: "clickable", which Mermaid adds to the classes of a node with a click, is a tag
// (scene.Item.Tags), and the rest are the author's, in their order.
func Clickable(classes []string) (class, tags []string) {
	for _, c := range classes {
		if c == "clickable" {
			tags = append(tags, c)
			continue
		}
		class = append(class, c)
	}
	return class, tags
}
