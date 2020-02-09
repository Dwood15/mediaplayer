package graphtraverse

import (
	"encoding/json"
	"io/ioutil"
)

type (
	NodeGraph map[NodeName]Node

	PlayerSimulator struct {
		Keys  map[KeyName]Key //Items which are considered "impossible" to remove from the pool
		Graph NodeGraph
	}
)

func LoadKeyPool(filename string) (keys []Key) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, &keys); err != nil {
		panic(err)
	}

	if len(keys) == 0 {
		panic("keys list is zero!!")
	}

	for _, k := range keys {
		if err = k.Validate(); err != nil {
			panic(err)
		}
	}

	return
}

func LoadNodeGraph(filename string) (ng NodeGraph) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, &ng); err != nil {
		panic(err)
	}

	if len(ng) == 0 {
		panic("keys list is zero!!")
	}

	for _, n := range ng {
		if err = n.Validate(); err != nil {
			panic(err)
		}
	}

	return
}

func ShuffleNodeGraph() NodeGraph {
	panic("notimplemented")

	return nil
}
