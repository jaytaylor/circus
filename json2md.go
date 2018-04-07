package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

    "github.com/Masterminds/sprig"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	Quiet   bool
	Verbose bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		errorExit(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "json2md",
	Short: "",
	Long:  "",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := os.MkdirAll("out", os.FileMode(int(0755))); err != nil {
			errorExit(fmt.Errorf("creating output directory: %s", err))
		}

		if err := convert(); err != nil {
			errorExit(err)
		}
	},
}

func convert() error {
	filenames, err := filepath.Glob("data/*.json")
	if err != nil {
		return err
	}

	for _, filename := range filenames {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading file %q: %s", filename, err)
		}

		context := map[string]interface{}{}
		// if err := json.Unmarshal(data, &context); err != nil {
		// 	return fmt.Errorf("parsing JSON from file %q: %s", filename, err)
		// }
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()

		if err := d.Decode(&context); err != nil {
			return fmt.Errorf("parsing JSON from file %q: %s", filename, err)
		}

		buf := &bytes.Buffer{}
		if err := mdTemplate.Execute(buf, context); err != nil {
			return fmt.Errorf("executing template for %q: %s", filename, err)
		}
		mdFilename := fmt.Sprintf("out/%v.md", context["ID"])
		if err := ioutil.WriteFile(mdFilename, buf.Bytes(), os.FileMode(int(0644))); err != nil {
			return fmt.Errorf("writing output file %q: %s", mdFilename, err)
		}
	}
	return nil
}

func errorExit(err interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}

func initLogging() {
	level := log.InfoLevel
	if Verbose {
		level = log.DebugLevel
	}
	if Quiet {
		level = log.ErrorLevel
	}
	log.SetLevel(level)
}

var mdTemplate = template.Must(template.New("md").Funcs(sprig.TxtFuncMap()).Parse(`---
title: {{ .Title | quote }}
date: {{ .Timestamp }}
draft: false
---

# {{ .Title }}

{{ .Timestamp }}

[Source]({{ .URL }})

ID: {{ .ID }}

Submitted by: [{{ .Submitter }}](https://news.ycombinator.com/user?id={{ .Submitter }})

[Discussion]({{ .CommentsURL }})

{{ .Goose.content }}
`))
