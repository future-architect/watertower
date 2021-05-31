# watertower

[![GoDoc](https://godoc.org/github.com/future-architect/watertower?status.svg)](https://godoc.org/github.com/future-architect/watertower)

![watertower](website/watertower.jpg)`

[© Copyright Steve Daniels and licensed under CC BY-SA 2.0](https://www.geograph.org.uk/photo/3103724)

Search Engine for Serverless environment.

* Search via words in documents
* Filter by tag

## Architecture

This API uses standard inverted index to achieve search. When searching, calculate score by TFIDF algorithm and sort.

This tool is using [gocloud.dev's docstore](https://gocloud.dev/howto/docstore/) as an storage.
So it can run completely managed environment (DynamoDB, Google Firestore, Azure CosmosDB) and MongoDB.

Elasticsearch can use flexible document, but this watertower can use only the following structure.
``"title"`` and ``"content"`` are processed (tokenize, stemming, remove stop words) by packages in ``github.com/future-architect/nlp``.
Now only English and Japanese are implemented.

```json
{
  "unique_key": "100",
  "lang": "en",
  "title": "100 Continue",
  "tags": ["100", "no-error"],
  "content": "This interim response indicates that everything so far is OK and that the client should continue the request, or ignore the response if the request is already finished.",
  "metadata": {"this is not":  "searchable, but you can store any strings"}
}
```

## Go API

Basically, this is an document storage. All API you should handle is ``WaterTower`` struct.
To create instance, you can pass URL.

Before using this search engine, you should import several packages:

```go
import (
    // Natural languages you want to use
	_ "github.com/future-architect/watertower/nlp/english"
	_ "github.com/future-architect/watertower/nlp/japanese"

    // The storage backends you want to use.
	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/awsdynamodb"
	_ "gocloud.dev/docstore/gcpfirestore"
	_ "gocloud.dev/docstore/mongodocstore"
)
```

Each storage should be initialized table with "id" as a primary key(partition key for DynamoDB).
``Index`` will be a table name.

```go
wt, err := watertower.NewWaterTower(ctx, watertower.Option{
    DocumentUrl: "dynamodb://", // default is "mem://"
    Index:       "docs",        // default is "index"
})
```

To store document, call ``PostDocument()`` method.

```go
docID, err := wt.PostDocument("unique-document-key", &watertower.Document{
    Language: "en",                               // required
    Title: "title",                               // document title
    Content: "content",                           // document body
    Tags: []string{"tag"},                        // tags
    Metadata: map[string]string{"extra": "data"}, // extra data
})
```

To get document, you can use by unique-key or document-id.

```go
doc, err := wt.FindDocumentByKey(uniqueKey)
docs, err := wt.FindDocuments(id1, id2)
```

To search document, use ``Search()`` method. First parameter is natural language search word for
title and content. Second parameter is tags to filter result. Third parameter is natural language name
to process search word.

```go
docs, err := wt.Search(searchWord, tags, lang)
```

To use local index file, open watertower instance without DocumentUrl option.

Then call ``ReadIndex(r io.Reader)`` for/and ``WriteIndex(w io.Writer)`` to read/store index:

```go
f, err := os.Open("watertower.idx")
if err != nil {
	panic(err)
}

wt, err := watertower.NewWaterTower(ctx, watertower.Option{
    DefaultLanguage: *defaultLanguage,
})
wt.ReadIndex(f)
```

### Sample Codes

#### httpstatus in /samples/httpstatus

CLI tool to search http status code. It reads from bundled documents and registeres them when booting.
It uses memdocstore as a backend datastore.

## HTTP interface

``watertower-server`` in ``/cmd/watertower-server`` implements Elasticsearch inspired API.

```sh
./watertower-server --port=8888
```

```sh
# Register document
$ curl -X POST "http://127.0.0.1:8888/index/_doc/"
　　-H "content-type: application/json"
　　-d '{ "unique_key": "id1", "title": "hello watertower",
　　　　　"content": "watertower is a full text search engine with tag filtering", "lang": "en" }'
{"_id":"d1","_index":"index","_type":"_doc","result":"created"}

# Get document by unique-key
$ curl -X GET "http://127.0.0.1:8888/index/_search?q=unique_key%3Aid1"
    -H"content-type: application/json"
{"hits":{"hits":[{"_id":"d1","_index":"index","_source":{"content":"watertower is a full text search engine with tag filtering","lang":"en","metadata":{},"tags":null,"title":"hello watertower","unique_key":"id1"},"_type":"_doc","sort":null}],"total":{"total":1}}}

# Get document by document ID
$ curl -X GET "http://127.0.0.1:8888/index/_source/d1"
(...)

# Search
$ curl -X GET "http://127.0.0.1:8888/index/_search"
  -H "content-type: application/json"
  -d '{"query": {"bool": {"must": {"match_phrase": {"content": {"query": "stay", "analyzer": "en"}}}}}}'
(...)
```

## CLI tool

CLI tool ``watertower-cli`` in ``/cmd/watertower-cli``.

```shell
$ ./watertower-cli --help
usage: watertower-cli [<flags>] <command> [<args> ...]

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).
  --default-language=DEFAULT-LANGUAGE  
          Default language

Commands:
  help [<command>...]
    Show help.

  create-index [<flags>] <INPUT>
    Generate Index

  search [<flags>] [<WORDS>...]
    Search
```

It can create single file index by using ``create-index`` sub command.

You can try searching the index file with ``search`` sub command.

## License

Apache2

