package restapi

import (
	"fmt"
	"github.com/future-architect/watertower"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/future-architect/watertower/webapi/models"
	"github.com/future-architect/watertower/webapi/restapi/operations"
	"github.com/go-openapi/runtime/middleware"
)

func getIndexDocID(params operations.GetIndexDocIDParams) middleware.Responder {
	wt, ok := watertowers[params.Index]
	if !ok {
		return operations.NewGetIndexDocIDNotFound().WithPayload(&operations.GetIndexDocIDNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	docID, err := strconv.ParseUint(params.ID[1:], 16, 32)
	if err != nil {
		return operations.NewGetIndexDocIDBadRequest().WithPayload(&operations.GetIndexDocIDBadRequestBody{
			Message: fmt.Sprintf("parse error id '%s' of index '%s'", params.ID, params.Index),
		})
	}
	docs, err := wt.FindDocuments(uint32(docID))
	if !ok {
		return operations.NewGetIndexDocIDNotFound().WithPayload(&operations.GetIndexDocIDNotFoundBody{
			ID:      params.ID,
			Index:   params.Index,
			Source:  nil,
			Type:    "_doc",
			Version: 0,
			Found:   false,
		})
	}
	doc := docs[0]
	return operations.NewGetIndexDocIDOK().WithPayload(&operations.GetIndexDocIDOKBody{
		ID:    params.ID,
		Index: params.Index,
		Source: &models.Document{
			Content:   doc.Content,
			Lang:      doc.Language,
			Metadata:  doc.Metadata,
			Tags:      doc.Tags,
			Title:     doc.Title,
			UniqueKey: doc.UniqueKey,
		},
		Type:    "_doc",
		Version: 0,
		Found:   true,
	})
}

func getIndexSourceID(params operations.GetIndexSourceIDParams) middleware.Responder {
	wt, ok := watertowers[params.Index]
	if !ok {
		return operations.NewGetIndexSourceIDNotFound().WithPayload(&operations.GetIndexSourceIDNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	docID, err := strconv.ParseUint(params.ID[1:], 16, 32)
	if err != nil {
		return operations.NewGetIndexSourceIDBadRequest().WithPayload(&operations.GetIndexSourceIDBadRequestBody{
			Message: fmt.Sprintf("parse error id '%s' of index '%s'", params.ID, params.Index),
		})
	}
	docs, err := wt.FindDocuments(uint32(docID))
	if !ok {
		return operations.NewGetIndexSourceIDNotFound().WithPayload(&operations.GetIndexSourceIDNotFoundBody{
			Message: fmt.Sprintf("document id '%s' in index '%s' is not found", params.ID, params.Index),
		})
	}
	doc := docs[0]
	return operations.NewGetIndexSourceIDOK().WithPayload(convertToResultDocument(doc))
}

func convertToResultDocument(doc *watertower.Document) *models.Document {
	return &models.Document{
		Content:   doc.Content,
		Lang:      doc.Language,
		Metadata:  doc.Metadata,
		Tags:      doc.Tags,
		Title:     doc.Title,
		UniqueKey: doc.UniqueKey,
	}
}

func getIndexSearch(params operations.GetIndexSearchParams) middleware.Responder {
	start := time.Now()
	wt, ok := watertowers[params.Index]
	if !ok {
		return operations.NewGetIndexDocIDNotFound().WithPayload(&operations.GetIndexDocIDNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	// search by unique_key
	if params.Q != nil {
		if !strings.HasPrefix(*params.Q, "unique_key:") {
			return operations.NewGetIndexSearchBadRequest().WithPayload(&operations.GetIndexSearchBadRequestBody{
				Message: "q= query only supports searching unique_key",
			})

		}
		uniqueKey := strings.TrimPrefix(*params.Q, "unique_key:")
		doc, err := wt.FindDocumentByKey(uniqueKey)
		duration := (time.Now().Sub(start) / time.Millisecond)
		if err != nil {
			return operations.NewGetIndexSearchOK().WithPayload(&operations.GetIndexSearchOKBody{
				Shards:   nil,
				Hits:     &operations.GetIndexSearchOKBodyHits{
					Hits:     []*operations.GetIndexSearchOKBodyHitsHitsItems0{},
					MaxScore: nil,
					Total:    &operations.GetIndexSearchOKBodyHitsTotal{
						Relation: "",
						Total:    0,
					},
				},
				TimedOut: false,
				Took:     int64(duration),
			})
		}
		return operations.NewGetIndexSearchOK().WithPayload(&operations.GetIndexSearchOKBody{
			Shards:   nil,
			Hits:     &operations.GetIndexSearchOKBodyHits{
				Hits:     []*operations.GetIndexSearchOKBodyHitsHitsItems0{
					{
						ID:     doc.ID,
						Index:  params.Index,
						Source: convertToResultDocument(doc),
						Type:   "_doc",
					},
				},
				Total:    &operations.GetIndexSearchOKBodyHitsTotal{
					Relation: "",
					Total:    1,
				},
			},
			TimedOut: false,
			Took:     int64(duration),
		})
	}
	var searchWord string
	var tags []string
	var lang string
	if params.Body.Query.Bool.Filter != nil {
		tags = params.Body.Query.Bool.Filter.Terms.Tags
	}
	if params.Body.Query.Bool.Must != nil {
		searchWord = *params.Body.Query.Bool.Must.MatchPhrase.Content.Query
		lang = params.Body.Query.Bool.Must.MatchPhrase.Content.Analyzer
	}
	docs, _ := wt.Search(searchWord, tags, lang)
	duration := (time.Now().Sub(start) / time.Millisecond)
	resultDocs  := []*operations.GetIndexSearchOKBodyHitsHitsItems0{}
	var maxScore float64
	for _, doc := range docs {
		resultDocs = append(resultDocs, &operations.GetIndexSearchOKBodyHitsHitsItems0{
			ID:     doc.ID,
			Index:  params.Index,
			Score:  doc.Score,
			Source: convertToResultDocument(doc),
			Type:   "_doc",
		})
		maxScore = math.Max(maxScore, doc.Score)
	}
	return operations.NewGetIndexSearchOK().WithPayload(&operations.GetIndexSearchOKBody{
		Shards:   nil,
		Hits:     &operations.GetIndexSearchOKBodyHits{
			Hits:     resultDocs,
			MaxScore: maxScore,
			Total: &operations.GetIndexSearchOKBodyHitsTotal{
				Relation: "",
				Total:    int64(len(docs)),
			},
		},
		TimedOut: false,
		Took:     int64(duration),
	})
}

func putIndexDoc(params operations.PutIndexDocIDParams) middleware.Responder {
	_, ok := watertowers[params.Index]
	if !ok {
		return operations.NewPutIndexDocIDNotFound().WithPayload(&operations.PutIndexDocIDNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	_, err := strconv.ParseUint(params.ID[1:], 16, 32)
	if err != nil {
		return operations.NewPutIndexDocIDBadRequest().WithPayload(&operations.PutIndexDocIDBadRequestBody{
			Message: fmt.Sprintf("parse error id '%s' of index '%s'", params.ID, params.Index),
		})
	}
	return middleware.NotImplemented("operation putIndexDocID has not yet been implemented")
}

func postIndexDoc(params operations.PostIndexDocParams) middleware.Responder {
	wt, ok := watertowers[params.Index]
	if !ok {
		return operations.NewPostIndexDocNotFound().WithPayload(&operations.PostIndexDocNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	metadata := make(map[string]string)
	if metamap, ok := params.Body.Metadata.(map[string]interface{}); ok {
		for key, value := range metamap {
			metadata[key] = fmt.Sprintf("%v", value)
		}
	}
	doc :=  &watertower.Document{
		Language: params.Body.Lang,
		Title: params.Body.Title,
		Content: params.Body.Content,
		Tags:   params.Body.Tags,
		Metadata: metadata,
	}
	numDocID, err := wt.PostDocument(params.Body.UniqueKey, doc)
	if err != nil {
		return operations.NewPostIndexDocInternalServerError().WithPayload(&operations.PostIndexDocInternalServerErrorBody{
			Message: fmt.Sprintf("Post document error on index '%s': %v", params.Index, err),
		})
	}
	docID := "d" + strconv.FormatUint(uint64(numDocID), 16)
	return operations.NewPostIndexDocOK().WithPayload(&models.ModifyResponse{
		ID:          docID,
		Index:       params.Index,
		PrimaryTerm: 0,
		SeqNo:       0,
		Shards:      nil,
		Type:        "_doc",
		Version:     0,
		Result:      "created",
	})
}

func deleteIndexDocID(params operations.DeleteIndexDocIDParams) middleware.Responder {
	wt, ok := watertowers[params.Index]
	if !ok {
		return operations.NewDeleteIndexDocIDNotFound().WithPayload(&operations.DeleteIndexDocIDNotFoundBody{
			Message: fmt.Sprintf("index '%s' is not found", params.Index),
		})
	}
	docID, err := strconv.ParseUint(params.ID, 10, 32)
	if err != nil {
		return operations.NewDeleteIndexDocIDBadRequest().WithPayload(&operations.DeleteIndexDocIDBadRequestBody{
			Message: fmt.Sprintf("parse error id '%s' of index '%s'", params.ID, params.Index),
		})
	}
	err = wt.RemoveDocumentByID(uint32(docID))
	if err != nil {
		return operations.NewDeleteIndexDocIDBadRequest().WithPayload(&operations.DeleteIndexDocIDBadRequestBody{
			Message: fmt.Sprintf("can't remove document id '%s' of index '%s': %v", params.ID, params.Index, err),
		})
	}
	return operations.NewDeleteIndexDocIDOK().WithPayload(&models.ModifyResponse{
		Index: params.Index,
		ID:    params.ID,
	})
}

