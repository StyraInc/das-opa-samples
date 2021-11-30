package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	ctx := context.Background()
	// TODO: update with your values (can be found in opa-conf.yaml from DAS)
	opa, err := setupOpa("https://kurt.styra.com/v1", <api token>, <system id>, ctx)
	if err != nil {
		panic(err)
	}
	defer opa.Stop(ctx)

	handleRequests()
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/something/{allow}", serveSomething)
	log.Fatal(http.ListenAndServe(":9099", myRouter))
}

// TODO: you likely want many more properties here
type Input struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Host    string            `json:"host"`
}

func serveSomething(w http.ResponseWriter, r *http.Request) {

	headers := map[string]string{}

	for name := range r.Header {
		headers[name] = r.Header.Get(name)
	}

	input := Input{
		Path:    r.URL.Path,
		Host:    r.Host,
		Headers: headers,
	}

	// path here matches your package/rule you have defined in DAS
	decision, err := callOpa("rules/main", input)
	if err != nil {
		panic(err)
	}

	// TODO: would normally look at the decision and return a 403 on deny
	// but for testing, I'm just retuning back the decision
	json.NewEncoder(w).Encode(decision)
}
