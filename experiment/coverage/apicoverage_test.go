/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"io"
	"regexp"
	"testing"
)

//NOTE: This method is for avoiding flake tests instead of using reflect.DeepEqual()
func equalAPIArray(a, b apiArray) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for _, i := range a {
		found := false
		for _, j := range b {
			if i.Method == j.Method && i.URL == j.URL {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestParseOpenAPI(t *testing.T) {
	testCases := []struct {
		Rawdata  []byte
		Expected apiArray
	}{
		{
			Rawdata: []byte(`{"paths": {"/resources": {
				"get": {"description": "get available resources"}}}}`),
			Expected: apiArray{
				{Method: "GET", URL: "/resources"},
			},
		},
		{
			Rawdata: []byte(`{"paths": {"/resources": {
				"get": {"description": "get available resources"},
				"post": {"description": "create resource"}}}}`),
			Expected: apiArray{
				{Method: "GET", URL: "/resources"},
				{Method: "POST", URL: "/resources"},
			},
		},
		{
			Rawdata: []byte(`{"paths": {
			"/resources": {
				"get": {"description": "get available resources"},
				"post": {"description": "create resource"}},
			"/foo": {
				"get": {"description": "get avaiable foo"},
				"post": {"description": "create foo"},
				"parameters": [{"type": "string", "description": "This should be ignored", "name": "bar", "in": "query"}]}}}`),
			Expected: apiArray{
				{Method: "GET", URL: "/resources"},
				{Method: "POST", URL: "/resources"},
				{Method: "GET", URL: "/foo"},
				{Method: "POST", URL: "/foo"},
			},
		},
	}
	for _, test := range testCases {
		res := parseOpenAPI(test.Rawdata)
		if !equalAPIArray(res, test.Expected) {
			t.Errorf("OpenAPI did not match expected for test")
			t.Errorf("%#v", res)
			t.Errorf("%#v", test.Expected)
		}
	}
}

func TestParseAPILog(t *testing.T) {
	testCases := []struct {
		Rawdata  io.Reader
		Expected apiArray
	}{
		{
			Rawdata: bytes.NewReader(
				[]byte(`
I0919 15:34:14.943642    6611 round_trippers.go:414] GET https://k8s-api/api/v1/foo
I0919 15:34:16.943642    6611 round_trippers.go:414] POST https://k8s-api/api/v1/bar
`)),
			Expected: apiArray{
				{Method: "GET", URL: "/api/v1/foo"},
				{Method: "POST", URL: "/api/v1/bar"},
			},
		},
		{
			Rawdata: bytes.NewReader(
				[]byte(`
I0919 15:34:14.943642    6611 round_trippers.go:414] GET https://k8s-api/api/v1/foo?other
`)),
			Expected: apiArray{
				{Method: "GET", URL: "/api/v1/foo"},
			},
		},
	}
	for _, test := range testCases {
		res := parseAPILog(test.Rawdata)
		if !equalAPIArray(res, test.Expected) {
			t.Errorf("APILog did not match expected for test")
			t.Errorf("Actual: %#v", res)
			t.Errorf("Expected: %#v", test.Expected)
		}
	}
}

func TestGetTestedAPIsByLevel(t *testing.T) {
	testCases := []struct {
		Negative       bool
		Reg            *regexp.Regexp
		apisOpenapi    apiArray
		apisTested     apiArray
		ExpectedTested apiArray
		ExpectedAll    apiArray
	}{
		{
			//Test Alpha APIs are returned
			Negative: false,
			Reg:      reAlphaAPI,
			apisOpenapi: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "POST", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "POST", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
				{Method: "POST", URL: "/apis/resources/v1beta1/"},
			},
			apisTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
			},
			ExpectedTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
			},
			ExpectedAll: apiArray{
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "POST", URL: "/apis/resources/v2alpha1/"},
			},
		},
		{
			//Test Beta APIs are returned
			Negative: false,
			Reg:      reBetaAPI,
			apisOpenapi: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "POST", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "POST", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
				{Method: "POST", URL: "/apis/resources/v1beta1/"},
			},
			apisTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
			},
			ExpectedTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
			},
			ExpectedAll: apiArray{
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
				{Method: "POST", URL: "/apis/resources/v1beta1/"},
			},
		},
		{
			//Test Stable APIs are returned
			Negative: true,
			Reg:      reNotStableAPI,
			apisOpenapi: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "POST", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "POST", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
				{Method: "POST", URL: "/apis/resources/v1beta1/"},
			},
			apisTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "GET", URL: "/apis/resources/v2alpha1/"},
				{Method: "GET", URL: "/apis/resources/v1beta1/"},
			},
			ExpectedTested: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
			},
			ExpectedAll: apiArray{
				{Method: "GET", URL: "/apis/resources/v1/"},
				{Method: "POST", URL: "/apis/resources/v1/"},
			},
		},
	}
	for _, test := range testCases {
		resTested, resAll := getTestedAPIsByLevel(test.Negative, test.Reg, test.apisOpenapi, test.apisTested)
		if !equalAPIArray(resTested, test.ExpectedTested) {
			t.Errorf("resTested did not match expected for test")
			t.Errorf("Expected: %#v", test.ExpectedTested)
			t.Errorf("Actual: %#v", resTested)
		}
		if !equalAPIArray(resAll, test.ExpectedAll) {
			t.Errorf("resAll did not match expected for test")
			t.Errorf("Expected: %#v", test.ExpectedAll)
			t.Errorf("Actual: %#v", resAll)
		}
	}
}
