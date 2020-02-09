package graphtraverse

import (
	"fmt"
)

type (
	NodeName  string //NodeName is the human-readable name of the node
	NodeClass string //NodeClass represents a category of node

	KeyName      string
	KeyCondition string //KeyCondition represents a requirement for using an item. A KeyCondition is either can_act, or the name of another key
	KeyAction    string //KeyAction indicates what to do after use of the key

	Action string //Action represents what to do when this node is visited

	//helper collections to make searching through them easier
	NodeClasses []NodeClass
	Actions     []Action
	KeyActions  []KeyAction
)

const (
	OneWayPortal NodeClass = "one_way_portal" // Blue Warps and Owl teleport
	TwoWayPortal NodeClass = "two_way_portal" // Doors, keyed entrances
	SingleGive   NodeClass = "single_give"    // Chests, GS, freestanding items
	ToggleGive   NodeClass = "toggle_give"    // Child -> Adult, visa versa

	Give            Action = "give"              // A Give action indicates that the player will receive an item
	Teleport        Action = "teleport"          // A Teleport Action says that the player should be teleported
	GiveAndTeleport Action = "give_and_teleport" // Visiting this node means player is teleported AND given item(s)

	OnUseDoNothing KeyAction = "do_nothing"
	OnUseDecrement KeyAction = "decrement"
	OnUseTeleport  KeyAction = "teleport_to"
)

type (
	OnVisit struct {
		Action      Action
		Gives       []KeyName //Gives is a list of Human-Readable items
		TeleportsTo string
	}

	Node struct {
		Name     NodeName  // Name is the human-readable identifier of the particular Node.
		Class    NodeClass // Class is a descriptor of the node
		Requires []KeyName // Names of the Items that are required in order to visit this node. ALL items in this list are required.
		OnVisit  OnVisit
		Exits    []string
	}

	// Key represents game state, or player save file state. Anything that can be used to indicate progression, really.
	Key struct {
		Name       KeyName        // Name is the human-readable ID of this key.
		Type       string         // Type is an extra descriptor for a key that can be added in lieu of listing all required items at once
		Conditions []KeyCondition // Conditions is a list of requirements in order to use this item. Expexts a KeyName
		State      struct {
			Action     KeyAction // Action: What to do on use of this key
			TeleportTo NodeName  // TeleportTo: Node to visit. Only valid if Action is teleport
			Value      int       // Value: the current number of this key in inventory
		}
	}

	//TODO: Move back to graph.go
	//I have no idea what I'm doing here.. this isn't a graph, more like a pre-graph :V
	NodeGraph map[NodeName]Node

	PlayerSimulator struct {
		Keys  map[KeyName]Key //Items which the player has in their inventory at this time.
		Graph NodeGraph
	}
)

//Implementation-detail stubs

var AllNodeClasses = NodeClasses{OneWayPortal, TwoWayPortal, SingleGive, ToggleGive}
var AllActions = Actions{Give, Teleport, GiveAndTeleport}
var AllKeyActions = KeyActions{OnUseDecrement, OnUseDoNothing, OnUseTeleport}

//Major helper funcs
func (n *Node) CanVisit(keysHeld map[KeyName]Key) bool {
	//idea: return all the items which are missing?

	for _, req := range n.Requires {
		k, ok := keysHeld[req]
		if !ok || len(k.Name) == 0 {
			return false
		}

		if !k.Use(keysHeld) {
			return false
		}

		//golang's funky about modifying members of a map...
		//I'm a scrub so we reassign it back to the map
		keysHeld[req] = k
	}

	return true
}

func (k *Key) Use(otherKeys map[KeyName]Key) (success bool) {
	if len(k.Conditions) == 0 {
		goto act
	}

	for _, condKey := range k.Conditions {
		if condKey == "can_act" {
			continue
		}

		//This bit here assumes that in order to use one key, we just have to have met the other key, _not_ used it.
		otherKey, ok := otherKeys[KeyName(condKey)]
		if !ok || otherKey.Validate() != nil {
			return false
		}
	}

act:
	if len(k.State.Action) == 0 {
		panic("invalid action: empty string")
	}

	if k.State.Action == OnUseDoNothing {
		return true
	}

	if k.State.Action == OnUseDecrement {
		if k.State.Value <= 0 {
			return false
		}

		k.State.Value--
		return true
	}

	//This shouldn't happen, I think?
	return false
}

//Basic sanity checks
func (k *Key) Validate() error {
	if len(k.Name) == 0 {
		return fmt.Errorf("all keys must have a name")
	}

	if !AllKeyActions.Contains(string(k.State.Action)) {
		return fmt.Errorf("key action [%s] is in valid must be from predeclared list", k.State.Action)
	}

	if k.State.Action == OnUseTeleport && len(k.State.TeleportTo) == 0 {
		return fmt.Errorf("TeleportTo must be ")
	}

	return nil
}

func (oV *OnVisit) Validate() error {
	if !AllActions.Contains(string(oV.Action)) {
		return fmt.Errorf("OnVisit invalid action type: [%s]", oV.Action)
	}

	if oV.Action == Give || oV.Action == GiveAndTeleport {
		if len(oV.Gives) == 0 {
			return fmt.Errorf("oV action: [%s] Gives item, but none were found in Gives list", oV.Action)
		}
	}

	if oV.Action == Teleport || oV.Action == GiveAndTeleport {
		if len(oV.TeleportsTo) == 0 {
			return fmt.Errorf("oV action: [%s] Teleports, but does not find any in Teleport list", oV.Action)
		}
	}

	return nil
}

func (n *Node) Validate() error {
	if len(n.Name) == 0 {
		return fmt.Errorf("node has no name. cannot use for tree traversal")
	}

	if !AllNodeClasses.Contains(string(n.Class)) {
		return fmt.Errorf("node class: [%s]", n.Class)
	}

	if n.Class == ToggleGive {
		panic("not implemented yet!")
	}

	//TODO: More validation of nodes for sanity checking

	return n.OnVisit.Validate()
}

//Minor helper-funcs

//The major issue with golang: no nice generics. :eye_roll:
func (nc NodeClasses) Contains(n string) bool {
	for _, v := range nc {
		if string(v) == n {
			return true
		}
	}

	return false
}

func (a Actions) Contains(n string) bool {
	for _, v := range a {
		if string(v) == n {
			return true
		}
	}

	return false
}

func (a KeyActions) Contains(n string) bool {
	for _, v := range a {
		if string(v) == n {
			return true
		}
	}

	return false
}
