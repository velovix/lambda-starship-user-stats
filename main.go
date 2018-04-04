// Exposes a REST API that stores usage statistics from (lambda () starship) in
// Datastore.
package main

import (
	"encoding/json"
	"net/http"

	"github.com/velovix/lambda-starship-user-stats/datatypes"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	_ "google.golang.org/appengine/remote_api"
)

func main() {
	http.Handle("/repl-command", postOnly(newREPLCommandHandler))
	http.Handle("/editor-content", postOnly(newEditorContentHandler))
	http.Handle("/error", postOnly(newErrorHandler))

	appengine.Main()
}

// newREPLCommandHandler stores a replCommand in datastore based on the data
// from the request.
func newREPLCommandHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var content datatypes.REPLCommand
	json.NewDecoder(r.Body).Decode(&content)

	// Write to the datastore
	key := datastore.NewKey(ctx, datatypes.REPLCommandKind, "", 0, nil)
	if _, err := datastore.Put(ctx, key, &content); err != nil {
		log.Errorf(ctx, "could not write to datastore: %v", err)
		http.Error(w, "Could not save REPL command", 500)
		return
	}

	log.Infof(ctx, "Saved REPL command %v", content)

	if _, err := w.Write([]byte{}); err != nil {
		log.Errorf(ctx, "failed to send response: %v", err)
		return
	}
}

// newEditorContentHandler stores an editorContent in datastore based on the
// data from the request.
func newEditorContentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var content datatypes.EditorContent
	json.NewDecoder(r.Body).Decode(&content)

	// Write to the datastore
	key := datastore.NewKey(ctx, datatypes.EditorContentKind, "", 0, nil)
	if _, err := datastore.Put(ctx, key, &content); err != nil {
		log.Errorf(ctx, "could not write to datastore: %v", err)
		http.Error(w, "Could not save editor content", 500)
		return
	}

	log.Infof(ctx, "Saved editor content %v", content)

	if _, err := w.Write([]byte{}); err != nil {
		log.Errorf(ctx, "failed to send response: %v", err)
		return
	}
}

// newErrorHandler stores an errorInstance in datastore based on the data from
// the request.
func newErrorHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var content datatypes.ErrorInstance
	json.NewDecoder(r.Body).Decode(&content)

	// Write to the datastore
	key := datastore.NewKey(ctx, datatypes.ErrorInstanceKind, "", 0, nil)
	if _, err := datastore.Put(ctx, key, &content); err != nil {
		log.Errorf(ctx, "could not write to datastore: %v", err)
		http.Error(w, "Could not save error", 500)
		return
	}

	log.Infof(ctx, "Saved error %v", content)

	if _, err := w.Write([]byte{}); err != nil {
		log.Errorf(ctx, "failed to send response: %v", err)
		return
	}
}

// postOnly is a middleware handler which fails if a request is anything other
// than a POST.
func postOnly(main func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
				return
			}

			main(w, r)
		},
	)
}
