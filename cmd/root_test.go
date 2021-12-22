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
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCmd(t *testing.T) {
	ts := httptest.NewServer(nil)
	defer ts.Close()

	rootCmd := prepareRootCmd()
	rootCmd.SetArgs([]string{"-c", "1", ts.URL})
	b := bytes.NewBufferString("")

	rootCmd.SetOut(b)

	cobra.CheckErr(rootCmd.Execute())

	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(out), "0.0% loss") {
		t.Fatal("Expected no loss with a local server")
	}
}
