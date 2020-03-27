// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/future-architect/watertower"
	"github.com/go-openapi/swag"
	"net/http"
	"os"
	"strings"

	"github.com/future-architect/watertower/webapi/restapi/operations"
	_ "github.com/future-architect/watertower/nlp/english"
	_ "github.com/future-architect/watertower/nlp/japanese"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"

	_ "gocloud.dev/docstore/memdocstore"
	_ "gocloud.dev/docstore/awsdynamodb"
	_ "gocloud.dev/docstore/gcpfirestore"
	_ "gocloud.dev/docstore/mongodocstore"
)

//go:generate swagger generate server --target ../../webapi --name Watertower --spec ../../swagger.yaml

var watertowerOptions = struct {
	Indexes     string `long:"indexes" description:"Comma separated search index name. Default value is 'index' or WATERTOWER_INDEXES envvar"`
	DocumentUrl string `long:"document-url" description:"Document URLs like firestore://my-project/my-documents, dynamo://, mongo. Default value is 'mem://' or WATERTOWER_DOCUMENT_URL envvar."`
	DryRun      bool   `long:"dump-collection-urls" description:"Dump collection urls to be initialized before running server."`
}{}

var watertowers = map[string]*watertower.WaterTower{}

func configureFlags(api *operations.WatertowerAPI) {
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		{
			ShortDescription: "Watertower config",
			LongDescription:  "Watertower configuration to specify index and storage",
			Options:          &watertowerOptions,
		},
	}
}

func configureAPI(api *operations.WatertowerAPI) http.Handler {
	if watertowerOptions.Indexes == "" {
		watertowerOptions.Indexes = os.Getenv("WATERTOWER_INDEXES")
		if watertowerOptions.Indexes == "" {
			watertowerOptions.Indexes = "index"
		}
	}
	if watertowerOptions.DocumentUrl == "" {
		watertowerOptions.DocumentUrl = os.Getenv("WATERTOWER_DOCUMENT_URL")
		if watertowerOptions.DocumentUrl == "" {
			watertowerOptions.DocumentUrl = "mem://"
		}
	}
	indexes := strings.Split(watertowerOptions.Indexes, ",")
	if watertowerOptions.DryRun {
		for _, index := range indexes {
			url, err := watertower.DefaultCollectionURL(watertower.Option{
				DocumentUrl: watertowerOptions.DocumentUrl,
				Index:       index,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error for index '%s': %v\n", index, err)
			} else {
				fmt.Printf("URL for '%s': %s\n", index, url)
			}
		}
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	hasError := false
	for _, index := range indexes {
		url, err := watertower.DefaultCollectionURL(watertower.Option{
			DocumentUrl: watertowerOptions.DocumentUrl,
			Index:       index,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error for index '%s': %v\n", index, err)
		}
		wt, err := watertower.NewWaterTower(ctx, watertower.Option{
			DocumentUrl: watertowerOptions.DocumentUrl,
			Index:       index,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error at initializing index '%s' at %s: err\n", index, url, err)
			hasError = true
		}
		watertowers[index] = wt
	}
	if hasError {
		fmt.Fprintln(os.Stderr, "Could not start watertower")
		os.Exit(1)
	}
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.DeleteIndexDocIDHandler = operations.DeleteIndexDocIDHandlerFunc(deleteIndexDocID)
	api.GetIndexDocIDHandler = operations.GetIndexDocIDHandlerFunc(getIndexDocID)
	api.GetIndexSearchHandler = operations.GetIndexSearchHandlerFunc(getIndexSearch)
	api.GetIndexSourceIDHandler = operations.GetIndexSourceIDHandlerFunc(getIndexSourceID)
	api.PostIndexDocHandler = operations.PostIndexDocHandlerFunc(postIndexDoc)
	api.PutIndexDocIDHandler = operations.PutIndexDocIDHandlerFunc(putIndexDoc)

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {
		cancel()
	}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
