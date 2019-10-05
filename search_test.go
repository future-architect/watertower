package watertower

import (
	"context"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSearchEN(t *testing.T) {
	wt, err := NewWaterTower(context.Background(), Option{
		CollectionPrefix: xid.New().String(),
		DocumentUrl:      "mem://",
	})
	assert.Nil(t, err)
	defer func() {
		err := wt.Close()
		assert.Nil(t, err)
	}()
	for _, data := range searchData {
		data.Language = "en"
		_, err := wt.PostDocument(data.UniqueKey, data)
		assert.Nil(t, err)
		if err != nil {
			return
		}
	}
	testcases := []struct {
		name       string
		searchWord string
		searchTag  []string
		found      bool
	}{
		{
			name:       "simple word search",
			searchWord: "post",
			searchTag:  nil,
			found:      true,
		},
		{
			name:       "simple tag search",
			searchWord: "",
			searchTag:  []string{"NoBody"},
			found:      true,
		},
		{
			name:       "word and tag search",
			searchWord: "post",
			searchTag:  []string{"200"},
			found:      true,
		},
		{
			name:       "word and tag conflict",
			searchWord: "post",
			searchTag:  []string{"NoBody"},
			found:      false,
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			docs, err := wt.Search(testcase.searchWord, testcase.searchTag, "en")
			assert.Nil(t, err)
			if testcase.found {
				assert.NotEmpty(t, docs)
			} else {
				assert.Empty(t, docs)
			}
		})
	}
}

var searchData = []*Document{
	{
		Title:     "100 Continue",
		UniqueKey: "100 Continue",
		Content: `100 Continue

This interim response indicates that everything so far is OK and that the wt should continue the request, or ignore the response if the request is already finished.`,
		Tags: []string{"100", "NoBody"},
	},
	{
		Title:     "101 Switching Protocol",
		UniqueKey: "101 Switching Protocol",
		Content: `
101 Switching Protocol

This code is sent in response to an Upgrade request header from the wt, and indicates the protocol the server is switching to.`,
		Tags: []string{"101", "NoBody"},
	},
	{
		Title:     "102 Processing",
		UniqueKey: "102 Processing",
		Content: `102 Processing

This code indicates that the server has received and is processing the request, but no response is available yet.`,
		Tags: []string{"102", "NoBody", "WebDAV"},
	},
	{
		Title:     "103 Early Hints",
		UniqueKey: "103 Early Hints",
		Content: `103 Early Hints

This status code is primarily intended to be used with the Link header, letting the user agent start preloading resources while the server prepares a response.`,
		Tags: []string{"103", "NoBody"},
	},
	{
		Title:     "200 OK",
		UniqueKey: "200 OK",
		Content: `200 OK

The request has succeeded. The meaning of the success depends on the HTTP method:
* GET: The resource has been fetched and is transmitted in the message body.
* HEAD: The entity headers are in the message body.
* PUT or POST: The resource describing the result of the action is transmitted in the message body.
* TRACE: The message body contains the request message as received by the server`,
		Tags: []string{"200"},
	},
	{
		Title:     "201 Created",
		UniqueKey: "201 Created",
		Content: `201 Created

The request has succeeded and a new resource has been created as a result.
This is typically the response sent after POST requests, or some PUT requests.`,
		Tags: []string{"201"},
	},
	{
		Title:     "202 Accepted",
		UniqueKey: "202 Accepted",
		Content: `202 Accepted

The request has been received but not yet acted upon.
It is noncommittal, since there is no way in HTTP to later send an asynchronous response indicating the outcome of the request.
It is intended for cases where another process or server handles the request, or for batch processing.`,
		Tags: []string{"202"},
	},
}