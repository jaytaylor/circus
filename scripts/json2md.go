package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kljensen/snowball"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"jaytaylor.com/circus/domain"
	"jaytaylor.com/circus/pkg/textmanip"
)

var (
	Limit   int
	Quiet   bool
	Verbose bool
)

func init() {
	rootCmd.PersistentFlags().IntVarP(&Limit, "limit", "l", -1, "Limit processing to the first N items")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		errorExit(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "json2md [input-file-or-path] [output-path]",
	Short: "",
	Long:  "If input-path is a directory, all files matching *.json will be read.  Both input and output paths can be set to '-' to read/write from/to stdout",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
		if Limit < -1 {
			errorExit(errors.New("Invalid limit, must be an integer greater than -1"))
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if args[1] != "-" {
			if err := os.MkdirAll(args[1], os.FileMode(int(0755))); err != nil {
				errorExit(fmt.Errorf("creating output directory: %s", err))
			}
		}

		fi, err := os.Stat(args[0])
		if err != nil {
			if args[0] == "-" {
				err = convert(args[0], args[1])
			}
			if err != nil {
				errorExit(err)
			}
			return
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			err = doBatch(args[0], args[1])
		default:
			err = convert(args[0], args[1])
		}

		if err != nil {
			errorExit(err)
		}
	},
}

func doBatch(inputPath string, outputPath string) error {
	filenames, err := filepath.Glob(fmt.Sprintf("%v%v*.json", inputPath, string(os.PathSeparator)))
	if err != nil {
		return err
	}
	for i, filename := range filenames {
		if Limit > -1 && i >= Limit {
			log.WithField("limit", Limit).Debug("Max requested items reached")
			break
		}
		if err := convert(filename, outputPath); err != nil {
			return err
		}
	}
	return nil
}

func convert(filename string, outputPath string) error {
	var (
		data []byte
		err  error
	)

	if filename == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(filename)
	}

	if err != nil {
		return fmt.Errorf("reading file %q: %s", filename, err)
	}

	context := &domain.Context{}
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

	if outputPath == "-" {
		fmt.Print(buf.String())
	} else {
		mdFilename := fmt.Sprintf("%v%v%v.md", outputPath, string(os.PathSeparator), context.ID)
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

var (
	entForbiddenExpr = regexp.MustCompile(`^(?:[0-9.]+|-+)$`)
	entMustExpr      = regexp.MustCompile(`^[a-zA-Z0-9 #$_'.,/-]+$`)
)

// cleanedEnts is a text template function which cleans and filters out
// suspicious NE's as well as hydrating in the stemmed field value.
func cleanedEnts(nes domain.NamedEntities) domain.NamedEntities {
	out := domain.NamedEntities{}
	for _, ne := range nes {
		ne.Entity = strings.Trim(ne.Entity, "\r\n\t ")
		if len(ne.Entity) == 0 {
			continue
		}

		name := textmanip.ToASCII(ne.Entity)
		stemmed, err := snowball.Stem(name, "english", true)
		if err != nil {
			log.Warnf("Unexpected error stemming %q: %s", ne.Entity, err)
		} else {
			ne.Stemmed = strings.Replace(stemmed, " ", "_", -1)
		}

		if len(name) >= 50 || entForbiddenExpr.MatchString(name) || !entMustExpr.MatchString(name) {
			log.Debugf("Skipping invalid named entity %q", name)
			continue
		}
		out = append(out, ne)
	}
	// Sort by frequency desc.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Frequency > out[j].Frequency
	})
	return out
}

// minFreqEnts is a text template function which filters out entities below the
// specified frequency count.
func minFreqEnts(nes domain.NamedEntities, minFreq int) domain.NamedEntities {
	out := cleanedEnts(nes)
	for _, ne := range nes {
		if ne.Frequency >= minFreq {
			out = append(out, ne)
		}
	}
	return out
}

// topNEnts returns at most the top N frequent entities.
func topNEnts(nes domain.NamedEntities, n int) domain.NamedEntities {
	out := make(domain.NamedEntities, len(nes))
	copy(out, nes)
	// Sort by frequency desc.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Frequency > out[j].Frequency
	})
	if n > len(out) {
		n = len(out)
	}
	return out[0:n]
}

var tplUtils = template.FuncMap{
	"cleanedEnts": cleanedEnts,
	"minFreqEnts": minFreqEnts,
	"topNEnts":    topNEnts,
}

var mdTemplate = template.Must(template.New("md").Funcs(sprig.TxtFuncMap()).Funcs(tplUtils).Parse(`---
{{- $cleaned := cleanedEnts .Article.NamedEntities -}}
{{- $top3Cleaned := topNEnts $cleaned 3 -}}

title: {{ .Title | quote }}
date: {{ .Timestamp }}
draft: false
{{- if gt (len $top3Cleaned) 0 }}
tags:
  {{- range $ne := $top3Cleaned }}
  - {{ $ne.Stemmed | quote }}
  {{- end }}
{{- end }}
---

ID: {{ .ID }}
|
[Discussion]({{ .CommentsURL }}) ({{ .Points }} point{{ if ne (printf "%v" .Points) (printf "%v" 1) }}s{{ end }}, {{ .Comments }} comment{{ if ne (printf "%v" .Comments) (printf "%v" 1) }}s{{ end }})
|
[Original Source]({{ .URL }})
|
Submitted by: [{{ .Submitter }}](https://news.ycombinator.com/user?id={{ .Submitter }})
|
Archives:
[archive.is](https://archive.is/{{ .URL }})
[archive.org](https://web.archive.org/web/*/{{ .URL }})

{{ if gt (len $cleaned) 0 -}}
Tags: {{ range $i, $ne := topNEnts $cleaned 10 }}{{ if gt $i 0 }}, {{ end }}[{{ $ne.Entity }}](/tags/{{ $ne.Stemmed }}){{ end }}
{{- end }}

{{ .Article.CleanedText }}
`))
