package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/shibukawa/watertower/nlp/english"
	_ "github.com/shibukawa/watertower/nlp/japanese"
	_ "gocloud.dev/docstore/memdocstore"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	tags  = kingpin.Flag("tag", "filter tag").Short('t').Strings()
	words = kingpin.Arg("search words", "word of document to search").Strings()
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wt, err := initWaterTower(ctx)
	if err != nil {
		panic(err)
	}
	defer wt.Close()

	kingpin.Parse()

	// color.Blue("words: %s", strings.Join(*words, " "))
	// color.Blue("tags: %s", strings.Join(*tags, ","))

	searchWord := strings.Join(*words, " ")
	if len(*tags) == 0 && searchWord == "" {
		kingpin.Usage()
		os.Exit(1)
	}

	docs, err := wt.Search(searchWord, *tags, "en")
	if len(docs) == 0 {
		color.Cyan("No Match")
	}
	for i, doc := range docs {
		color.Green("\n%d: -----------------------------------------------------------\n", i+1)

		fmt.Println(doc.Content)

		if len(docs)-1 == i {
			color.Green("\n---------------------------------------------------------------\n")
		}
	}
}
