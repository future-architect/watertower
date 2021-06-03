package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/future-architect/watertower"
	_ "github.com/future-architect/watertower/nlp/english"
	_ "github.com/future-architect/watertower/nlp/japanese"
)

type Query struct {
	Query   string              `json:"query"`
	Lang    string              `json:"lang"`
	Tags    []string            `json:"tags"`
}

type Result struct {
	Error  string                 `json:"error:,omitempty"`
	Count  int                    `json:"count,omitempty"`
	Result []*watertower.Document `json:"result,omitempty"`
}

func errorResult(status int, err string) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(&Result{
		Error: err,
	})
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(b),
	}
}

var (
	indexes         string
	filePath        string
	documentUrl     string
	defaultLanguage string
)

func init() {
	indexes = os.Getenv("WATERTOWER_INDEXES")
	if indexes == "" {
		indexes = "index"
	}
	documentUrl = os.Getenv("WATERTOWER_DOCUMENT_URL")
	if strings.HasPrefix(documentUrl, "file://") {
		filePath = filepath.FromSlash(strings.TrimPrefix(documentUrl, "file://"))
		documentUrl = ""
	} else if documentUrl == "" {
		filePath = "watertower.idx"
	}
	if defaultLanguage == "" {
		defaultLanguage = os.Getenv("WATERTOWER_DEFAULT_LANGUAGE")
		if defaultLanguage == "" {
			defaultLanguage = "en"
		}
	}
}

func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var query Query
	json.Unmarshal([]byte(request.Body), &query)
	for key, values := range request.MultiValueQueryStringParameters {
		if key == "query" {
			query.Query = strings.Join(values, " ")
		} else if key == "tag" {
			query.Tags = values
		} else if key == "lang" {
			query.Lang = strings.Join(values, ",")
		}
	}

	wt, err := watertower.NewWaterTower(ctx, watertower.Option{
		DefaultLanguage: defaultLanguage,
		DocumentUrl:     documentUrl,
	})

	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return errorResult(500,fmt.Sprintf("file open error: %v", err)), nil
		}
		defer f.Close()
		err = wt.ReadIndex(f)
		if err != nil {
			return errorResult(500,fmt.Sprintf("parse index file error: %v", err)), nil
		}
	}

	lang := query.Lang
	if lang == "" {
		lang = defaultLanguage
	}
	docs, err := wt.Search(query.Query, query.Tags, lang)
	if err != nil {
		return errorResult(500,fmt.Sprintf("search error: %v", err)), nil
	}

	b, _ := json.Marshal(&Result{
		Count:  len(docs),
		Result: docs,
	})
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body: string(b),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
