// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee/pkg/api"
	"github.com/ethersphere/bee/pkg/logging"
	"github.com/ethersphere/bee/pkg/manifest"
	"github.com/ethersphere/bee/pkg/pingpong"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/tags"
	"resenje.org/web"
)

type testServerOptions struct {
	Pingpong       pingpong.Interface
	Storer         storage.Storer
	ManifestParser manifest.Parser
	Tags           *tags.Tags
	Logger         logging.Logger
}

func newTestServer(t *testing.T, o testServerOptions) *http.Client {
	if o.Logger == nil {
		o.Logger = logging.New(ioutil.Discard, 0)
	}
	s := api.New(api.Options{
		Tags:           o.Tags,
		Storer:         o.Storer,
		ManifestParser: o.ManifestParser,
		Logger:         o.Logger,
	})
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)

	return &http.Client{
		Transport: web.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			u, err := url.Parse(ts.URL + r.URL.String())
			if err != nil {
				return nil, err
			}
			r.URL = u
			return ts.Client().Transport.RoundTrip(r)
		}),
	}
}
