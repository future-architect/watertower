package main

import (
	"context"
	"encoding/json"
	"github.com/shibukawa/watertower"
	"go.pyspa.org/brbundle"
)

func initWaterTower(ctx context.Context) (wt *watertower.WaterTower, err error) {
	wt, err = watertower.NewWaterTower(ctx, watertower.Option{
		DocumentUrl: "mem://",
	})
	if err != nil {
		return nil, err
	}
	fileNames := brbundle.DefaultRepository.FilesInDir("")
	for _, fileName := range fileNames {
		file, err := brbundle.DefaultRepository.Find(fileName)
		if err != nil {
			return nil, err
		}
		reader, err := file.Reader()
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(reader)
		var document watertower.Document
		err = decoder.Decode(&document)
		reader.Close()
		if err != nil {
			return nil, err
		}
		_, err = wt.PostDocument(document.UniqueKey, &document)
		if err != nil {
			return nil, err
		}
	}
	return wt, nil
}
