// Copyright 2021 Raphaël P. Barazzutti
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"fever.ch/http-ping/app"
	"io"
	"io/ioutil"
	"testing"
)

type httpPingMockBuilder struct {
	config *app.Config
}

type httpPingMock struct{}

func (httpPingMock *httpPingMock) Run() error {
	return nil
}

func (httpPingMockBuilder *httpPingMockBuilder) newHTTPPingMock(config *app.Config, _ io.Writer) (app.HTTPPing, error) {
	httpPingMockBuilder.config = config
	return &httpPingMock{}, nil
}

func commandTest(_ *testing.T, args []string) (*app.Config, []byte, error) {

	builder := httpPingMockBuilder{}
	rootCmd := prepareRootCmd(builder.newHTTPPingMock)

	rootCmd.SetArgs(args)
	b := bytes.NewBufferString("")

	rootCmd.SetOut(b)
	//
	err := rootCmd.Execute()
	if err != nil {
		return nil, nil, err
	}

	out, err := ioutil.ReadAll(b)

	if err != nil {
		return nil, nil, err
	}

	return builder.config, out, nil
}

func TestCount(t *testing.T) {
	config, _, err := commandTest(t, []string{"-c", "123", "www.google.com"})
	if err != nil || config.Count != 123 {
		t.Fatal("Count parameter not taken in account")
	}
}

func TestEnforceIPv6(t *testing.T) {
	config, _, err := commandTest(t, []string{"-6", "www.google.com"})
	if err != nil && config.IPProtocol != "ip6" {
		t.Fatal("IPv6 parameter not taken in account")
	}

}

func TestEnforceIPv4AndIPv6(t *testing.T) {
	_, _, err := commandTest(t, []string{"-4", "-6", "www.google.com"})
	if err == nil {
		t.Fatal("IPv6 parameter not taken in account")
	}

}

func TestExcessOfArguments(t *testing.T) {
	_, _, err := commandTest(t, []string{"www.google.com", "www.wikipedia.com"})
	if err == nil {
		t.Fatal("only one non-flag argument can we used")
	}

}

func TestLackOfArguments(t *testing.T) {
	_, _, err := commandTest(t, []string{})
	if err == nil {
		t.Fatal("only one non-flag argument can we used")
	}
}

func TestCookie(t *testing.T) {
	config, _, err := commandTest(t, []string{"--cookie", "animal=cat", "www.google.com"})
	if err != nil || len(config.Cookies) != 1 || config.Cookies[0].Name != "animal" || config.Cookies[0].Value != "cat" {
		t.Fatal("cookie flag not taken in account")
	}
}
