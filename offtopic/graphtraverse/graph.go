package graphtraverse

type (
	//I have no idea what I'm doing here.. this isn't a graph, more like a pre-graph :V
	NodeGraph map[NodeName]Node

	PlayerSimulator struct {
		Keys  map[KeyName]Key //Items which the player has in their inventory at this time.
		Graph NodeGraph
	}
)

func ShuffleNodeGraph() NodeGraph {
	panic("notimplemented")

	return nil
}
