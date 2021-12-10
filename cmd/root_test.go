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
