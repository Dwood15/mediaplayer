package graphtraverse

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

const (
	interior        = "Interior"
	specialInterior = "SpecialInterior"
	grotto          = "Grotto"
	grave           = "Grave"
	specialGrave    = "SpecialGrave"
	overworld       = "Overworld"
	owlDrop         = "OwlDrop"
)

type indexmap map[string]string
type EntranceExit []map[string]map[string]indexmap

type EntranceExitPair struct {
	Type     string
	Entrance map[string]uintptr
	Exit     map[string]uintptr
}

func loadFile() {
	entranceList := EntranceExit{}
	ens, err := os.Open("entrance_list.json5")
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadAll(ens)
	if err != nil {
		panic(err)
	}

	if len(b) == 0 {
		panic("file failed to load!")
	}

	bs := string(b)
	_ = bs
	//decoder := json.NewDecoder(ens)
	//decoder.UseNumber()
	err = json.Unmarshal(b, &entranceList)
	if err != nil {
		panic(err)
	}

	if len(entranceList) == 0 {
		panic("entranceList empty!!")
	}
}
