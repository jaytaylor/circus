package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gigawattio/oslib"
	goose "github.com/jaytaylor/GoOse"
	"github.com/spf13/cobra"
)

const NLPWebAddr = "127.0.0.1:8000"

// NamedEntity represents a named-entity as it relates to a document.
type NamedEntity struct {
	Frequency int    `json:"frequency"` // Number of occurrences in document.
	Entity    string `json:"entity"`    // String content value of entity.
	Label     string `json:"label"`     // Category of named entity.
	POS       string `json:"pos"`       // Part-of-speech.
}

type NamedEntities []NamedEntity

var (
	// Favorites string
	Quiet   bool
	Verbose bool
)

func init() {
	rootCmd.AddCommand(versionCmd)
	// rootCmd.PersistentFlags().StringVarP(&Favorites, "favorites", "favs", "", "favorites.json file (`hn-utils' will be run when not provided")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		errorExit(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hydrator",
	Short: "",
	Long:  "",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			g       = goose.New()
			article *goose.Article
			err     error
		)

		if args[0] == "-" {
			bs, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				errorExit(fmt.Errorf("reading stdin: %s", err))
			}
			article, err = g.ExtractFromRawHTML("", string(bs))
		} else {
			article, err = g.ExtractFromURL(args[0])
		}
		if err != nil {
			errorExit(fmt.Errorf("extracting article: %s", err))
		}
		// https://brandur.org/rust-web -o json > rust.json | jq -r '.content' < rust.json | curl 'http://127.0.0.1:8000/v1/named-entities?instance=lg' -d@- > ners.json

		asJSON, err := json.MarshalIndent(*article, "", "    ")
		if err != nil {
			errorExit(fmt.Errorf("marshalling article: %s", err))
		}

		asMap := map[string]interface{}{}
		if err := json.Unmarshal(asJSON, &asMap); err != nil {
			errorExit(fmt.Errorf("unmarshalling to map: %s", err))
		}

		if _, ok := asMap["content"]; !ok {
			errorExit(errors.New("no content found in article map"))
		}
		plaintext := asMap["content"].(string)

		nlpSig, nlpAck, err := launchNLPWeb()
		if err != nil {
			errorExit(fmt.Errorf("starting nlpweb.py: %s", err))
		}

		resp, err := http.Post(fmt.Sprintf("http://%v/v1/named-entities?instance=lg", NLPWebAddr), "application/x-www-form-urlencoded", bytes.NewBufferString(plaintext))
		if err != nil {
			errorExit(fmt.Errorf("submitting article to ner extractor: %s", err))
		}
		if resp.StatusCode/100 != 2 {
			errorExit(fmt.Errorf("article ner submission received non-2xx response status-code=%v", resp.StatusCode))
		}

		nesBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errorExit(fmt.Errorf("reading article ner submission body: %s", err))
		}
		if err := resp.Body.Close(); err != nil {
			errorExit(fmt.Errorf("closing article ner submission body: %s", err))
		}

		nes := NamedEntities{}
		if err := json.Unmarshal(nesBody, &nes); err != nil {
			errorExit(fmt.Errorf("unmarshalling named entities: %s", err))
		}

		asMap["named-entities"] = nes

		bs, err := json.MarshalIndent(&asMap, "", "    ")
		if err != nil {
			errorExit(fmt.Errorf("serializing final result: %s", err))
		}

		fmt.Println(string(bs))

		nlpSig <- os.Interrupt
		<-nlpAck
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information for this thing",
	Long:  "All software has versions. This is this the one for thing..",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("goose-cli HTML Content / Article extractor command-line interface v0.0")
	},
}

func launchNLPWeb() (chan os.Signal, chan struct{}, error) {
	cmd := exec.Command("/usr/bin/env", "bash", "-c",
		fmt.Sprintf(": && set -o errexit && cd %q && source venv/bin/activate && python nlpweb.py %v", oslib.PathDirName(os.Args[0]), NLPWebAddr),
	)

	{
		r, w := io.Pipe()
		cmd.Stdout = w
		go func() {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				log.Debugf("[nlpweb][stdout] %v", scanner.Text())
			}
		}()
	}

	{
		r, w := io.Pipe()
		cmd.Stderr = w
		go func() {
			scanner := bufio.NewScanner(r)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() {
				log.Debugf("[nlpweb][stderr] %v", scanner.Text())
			}
		}()
	}

	log.Debugf("Starting nlpweb.py on address=%v", NLPWebAddr)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	var (
		d = net.Dialer{
			Timeout: 1 * time.Second,
		}
		since   = time.Now()
		maxWait = 10 * time.Second
	)
	for {
		if time.Now().Sub(since) > maxWait {
			return nil, nil, fmt.Errorf("timed out after %s waiting for nlpweb.py to start", maxWait)
		}
		conn, err := d.Dial("tcp", NLPWebAddr)
		if err == nil {
			if err = conn.Close(); err != nil {
				return nil, nil, fmt.Errorf("unexpected error closing connection to nlpweb.py: %s", err)
			}
			break
		}
		log.Debug(".")
		time.Sleep(100 * time.Millisecond)
	}
	log.Debugf("Started nlpweb.py OK, pid=%v", cmd.Process.Pid)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, os.Kill)

	ack := make(chan struct{})

	go func() {
		<-sig // Wait for ^C signal.
		fmt.Fprintln(os.Stderr, "\nInterrupt or kill signal detected, shutting down..")

		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("Shutting down nlpweb.py: %s", err)
		}
		log.Debugf("Killed nlpweb.py") // out=%v err=%v", stdout.String(), stderr.String())

		select {
		case ack <- struct{}{}:
		default:
		}
	}()

	return sig, ack, nil
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
