package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/open-policy-agent/opa/sdk"
)

var OPA *sdk.OPA
var CTX context.Context

func setupOpa(baseUrl string, token string, systemId string, ctx context.Context) (*sdk.OPA, error) {

	CTX = ctx

	config := configByParameters(baseUrl, token, systemId)
	//config := configByFile("opa-conf.yaml")

	opa, err := sdk.New(ctx, sdk.Options{
		Config: bytes.NewReader(config),
	})
	if err != nil {
		return nil, err
	}
	OPA = opa

	return opa, nil
}

func configByFile(file string) []byte {

	config, err := os.ReadFile("opa-conf.yaml")
	if err != nil {

		panic(err)
	}
	return config
}

func configByParameters(baseUrl string, token string, systemId string) []byte {

	return []byte(fmt.Sprintf(`{
		"discovery": {
		  "name": "discovery",
		  "resource": "/systems/%v/discovery",
		  "service": "styra"
		},
		"labels": {
		  "system-id": %q,
		  "system-type": "custom"
		},
		"services": [
		  {
			"credentials": {
			  "bearer": {
				"token": %q
			  }
			},
			"name": "styra",
			"url": %q
		  }
		]
	  }`, systemId, systemId, token, baseUrl))
}

func callOpa(path string, input interface{}) (interface{}, error) {
	result, err := OPA.Decision(CTX, sdk.DecisionOptions{Path: path, Input: input})
	if err != nil {
		return nil, err
	}
	return result, nil
}
