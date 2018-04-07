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
	"strings"
	"time"

	"gigawatt.io/oslib"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	goose "jaytaylor.com/GoOse"
	"jaytaylor.com/archive.is"
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
	Quiet           bool
	Verbose         bool
	AltNLPWebServer string
)

func init() {
	rootCmd.AddCommand(versionCmd)
	// rootCmd.PersistentFlags().StringVarP(&Favorites, "favorites", "favs", "", "favorites.json file (`hn-utils' will be run when not provided")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&AltNLPWebServer, "nlpweb-server", "s", "", "Base URL to already running NLPWeb server (saves on the enormous overhead of launching and initializing one)")
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
		} else if strings.HasSuffix(strings.ToLower(args[0]), ".pdf") { // TODO: Make more robust, with a proper URL parse.
			article, err = handlePDF(args[0])
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
			// log.Debug(string(asJSON))
			if args[0] == "-" {
				errorExit(errors.New("no content found in article map"))
			} else {
				asMap, asJSON, article, err = archiveIsFallback(args[0])
				if err != nil {
					errorExit(fmt.Errorf("no content found in article map, and fallback error was: %s", err))
				}
				if _, ok := asMap["content"]; !ok {
					errorExit(errors.New("no content found in article map, even after applying archive.is fallback"))
				}
			}
		}
		plaintext := asMap["content"].(string)

		err = withNLPWeb(func(nlpWebURL string) error {
			u := fmt.Sprintf("%v/v1/named-entities?instance=lg", nlpWebURL)
			resp, err := http.Post(u, "application/x-www-form-urlencoded", bytes.NewBufferString(plaintext))
			if err != nil {
				return fmt.Errorf("submitting article to ner extractor: %s", err)
			}
			if resp.StatusCode/100 != 2 {
				return fmt.Errorf("article ner submission received non-2xx response status-code=%v", resp.StatusCode)
			}

			nesBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading article ner submission body: %s", err)
			}
			if err := resp.Body.Close(); err != nil {
				return fmt.Errorf("closing article ner submission body: %s", err)
			}

			nes := NamedEntities{}
			if err := json.Unmarshal(nesBody, &nes); err != nil {
				return fmt.Errorf("unmarshalling named entities: %s", err)
			}

			asMap["namedEntities"] = nes

			bs, err := json.MarshalIndent(&asMap, "", "    ")
			if err != nil {
				return fmt.Errorf("serializing final result: %s", err)
			}
			fmt.Println(string(bs))
			return nil
		})

		if err != nil {
			errorExit(err)
		}
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

func archiveIsFallback(url string) (map[string]interface{}, []byte, *goose.Article, error) {
	s, err := archiveis.Capture(url)
	if err != nil {
		return nil, nil, nil, err
	}
	article, err := goose.New().ExtractFromRawHTML(url, s)
	if err != nil {
		return nil, nil, nil, err
	}
	asJSON, err := json.MarshalIndent(*article, "", "    ")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshalling article: %s", err)
	}

	asMap := map[string]interface{}{}
	if err := json.Unmarshal(asJSON, &asMap); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshalling to map: %s", err)
	}
	return asMap, asJSON, article, nil
}

func handlePDF(url string) (*goose.Article, error) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("set -o errexit && set -o pipefail && set -o nounset && curl -sSL -o /tmp/pdf.pdf %q | gs -sDEVICE=txtwrite -o /tmp/pdf.txt /tmp/pdf.pdf", args[0]))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("converting PDF to txt: %s", err)
	}
	text, err := ioutil.ReadFile("/tmp/pdf.txt")
	article := &goose.Article{
		CleanedText: string(text),
	}
	return article, nil
}

func withNLPWeb(fn func(baseURL string) error) error {
	var baseURL string

	if AltNLPWebServer == "" {
		nlpSig, nlpAck, err := launchNLPWeb()
		if err != nil {
			return fmt.Errorf("starting nlpweb.py: %s", err)
		}

		defer func() {
			nlpSig <- os.Interrupt
			<-nlpAck
		}()

		baseURL = fmt.Sprintf("http://%v", NLPWebAddr)
	} else {
		baseURL = AltNLPWebServer
	}

	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = fmt.Sprintf("http://%v", baseURL)
	}

	err := fn(baseURL)

	if err != nil {
		return err
	}
	return nil
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
