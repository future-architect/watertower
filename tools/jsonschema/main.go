package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/jsonschema"
	"github.com/shibukawa/watertower"
	"os"
	"path/filepath"
	"runtime"
)

// https://gist.github.com/abrookins/2732551#file-gistfile1-go
func __FILE__() string {
	_, filepath, _, _ := runtime.Caller(1)
	return filepath
}

func __DIR__() string {
	return filepath.Dir(__FILE__())
}

func gen(fileName string, target interface{}) {
	prjPath := filepath.Join(__DIR__(), "../../", fileName)
	fmt.Printf("writing: %s\n", prjPath)
	prj, err := os.Create(prjPath)
	if err != nil {
		panic(err)
	}
	defer prj.Close()
	schema := jsonschema.Reflect(target)
	e := json.NewEncoder(prj)
	e.SetIndent("", "  ")
	e.Encode(schema)
}

func main() {
	gen("document-schema.json", &watertower.Document{})
}
