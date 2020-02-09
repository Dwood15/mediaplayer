package graphtraverse

import (
	"testing"
)

func TestLoadKeyPool(t *testing.T) {
	//sorry, windows users :P
	LoadKeyPool("test_fixtures/vanilla/keys.json")
}

func TestBuildNodeGraph(t *testing.T) {

}

//func TestLoadFile(t *testing.T) {
//	loadFile()
//}