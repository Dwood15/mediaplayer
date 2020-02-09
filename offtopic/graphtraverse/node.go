package graphtraverse

type NodeClass string
type NodeSubClass string

const (
	Entrance NodeClass = "entrance_node"
	Chest    NodeClass = "chest_node"
	Item     NodeClass = "item_node"
)

const (
	Locked   NodeSubClass = "locked"
	Unlocked NodeSubClass = "unlocked"
)

type Node struct {
	Name string //The identifier of the particular Node.

}

type Graph struct {

}