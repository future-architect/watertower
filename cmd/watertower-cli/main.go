package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/future-architect/watertower"
	_ "github.com/future-architect/watertower/nlp/english"
	_ "github.com/future-architect/watertower/nlp/japanese"
	_ "gocloud.dev/docstore/memdocstore"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	defaultLanguage = kingpin.Flag("default-language", "Default language").String()

	generateCmd   = kingpin.Command("create-index", "Generate Index")
	outputFile    = generateCmd.Flag("output", "Output file").File()
	forceLanguage = generateCmd.Flag("force-language", "Force language").String()
	inputFolder   = generateCmd.Arg("INPUT", "Input Folder").Required().ExistingDir()

	searchCmd   = kingpin.Command("search", "Search")
	inputFile   = searchCmd.Flag("input", "Input index file").File()
	tags        = searchCmd.Flag("tag", "Tags").Strings()
	language    = searchCmd.Flag("language", "Search language").String()
	searchWords = searchCmd.Arg("WORDS", "Search words").Strings()
)

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(time.Time{})
}

func generate(ctx context.Context) error {
	wt, err := watertower.NewWaterTower(ctx, watertower.Option{
		DefaultLanguage: *defaultLanguage,
	})
	if err != nil {
		return err
	}
	defer wt.Close()

	filepath.Walk(*inputFolder, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		dec := json.NewDecoder(f)
		var doc watertower.Document
		err = dec.Decode(&doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse file error: %s - %s\n", path, err.Error())
			return nil
		}
		if doc.UniqueKey == "" {
			doc.UniqueKey = path
		}
		if *forceLanguage != "" {
			doc.Language = *forceLanguage
		}
		_, err = wt.PostDocument(doc.UniqueKey, &doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "register document error: %s - %s\n", path, err.Error())
		}
		fmt.Printf("  adding %s (lang=%s, path=%s)\n", doc.Title, doc.Language, path)
		return nil
	})

	if *outputFile == nil {
		f, err := os.Create("watertower.idx")
		if err != nil {
			panic(err.Error())
		}
		outputFile = &f
	}
	defer (*outputFile).Close()

	err = wt.WriteIndex(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write index error: %s\n", err.Error())
	}
	return err
}

func search(ctx context.Context) error {
	var input io.Reader
	if *inputFile != nil {
		input = *inputFile
		defer (*inputFile).Close()
	} else {
		f, err := os.Open("watertower.idx")
		if err != nil {
			return err
		}
		defer f.Close()
		input = f
	}
	wt, err := watertower.NewReadOnlyWaterTower(ctx, watertower.ReadOnlyOption{
		DefaultLanguage: *defaultLanguage,
		Input:           input,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "open index error: %s\n", err.Error())
		return err
	}
	defer wt.Close()

	lang := *language
	if lang == "" {
		lang = *defaultLanguage
	}
	docs, err := wt.Search(strings.Join(*searchWords, " "), *tags, lang)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search error: %s\n", err.Error())
		return err
	}

	if len(docs) == 0 {
		color.Cyan("No Match")
	}
	for i, doc := range docs {
		if i != 0 {
			color.Green("\n-----------------------------------------------------------\n\n")
		}

		color.Blue("# %s \n\n", doc.Title)
		color.Cyan(doc.Content)
	}

	return nil
}

func main() {
	ctx := context.Background()

	switch kingpin.Parse() {
	case generateCmd.FullCommand():
		err := generate(ctx)
		if err != nil {
			os.Exit(1)
		}
	case searchCmd.FullCommand():
		err := search(ctx)
		if err != nil {
			os.Exit(1)
		}
	}
}
