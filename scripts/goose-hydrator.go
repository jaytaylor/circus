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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	goose "jaytaylor.com/GoOse"
	archiveis "jaytaylor.com/archive.is"
)

const NLPWebAddr = "127.0.0.1:8000"

const UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.162 Safari/537.36"

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
	RequestTimeout  time.Duration

	PDFProcessorTimeout = 30 * time.Second
)

func init() {
	rootCmd.AddCommand(versionCmd)
	// rootCmd.PersistentFlags().StringVarP(&Favorites, "favorites", "favs", "", "favorites.json file (`hn-utils' will be run when not provided")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
	rootCmd.PersistentFlags().StringVarP(&AltNLPWebServer, "nlpweb-server", "s", "", "Base URL to already running NLPWeb server (saves on the enormous overhead of launching and initializing one)")
	rootCmd.PersistentFlags().DurationVarP(&RequestTimeout, "http-timeout", "t", 10*time.Second, "HTTP timeout value when downloading HTML content")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		errorExit(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "hydrator",
	Short: "Identifies and tags main content in an HTML document",
	Long:  "Use GoOse and SpaCy to extract main page content and tag it with keywordsa",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			content []byte
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
		} else if strings.HasSuffix(strings.ToLower(args[0]), ".pdf") { // TODO: Make more robust, with a proper HTTP header content-type check.
			log.Warn("STILL NEED TO IMPLEMENT BIN DATA SUPPORT AND JUST SERVE UP THE ARBITRARY BIN CONTENT + APPROPRIATE HEADER.")
			log.Warn("---\nThere is still a lot to figure out between this and resurrecting deadlinks from archive.is and archive.org")
			if content, err = handlePDF(args[0]); err != nil {
				errorExit(fmt.Errorf("downloading and converting PDF to HTML: %s", err))
			}
			article, err = g.ExtractFromRawHTML(args[0], string(content))
		} else {
			if content, err = download(args[0], RequestTimeout); err != nil {
				errorExit(fmt.Errorf("downloading article: %s", err))
			}
			article, err = g.ExtractFromRawHTML(args[0], string(content))
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
		//errorExit(fmt.Errorf("%s", asMap))

		if _, ok := asMap["content"]; !ok {
			// log.Debug(string(asJSON))
			if args[0] == "-" {
				errorExit(errors.New("no content found in article map"))
			} else {
				asMap, asJSON, article, err = archiveIsFallback(args[0], RequestTimeout)
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
		fmt.Println("jay's hydrator thingamajig")
	},
}

func archiveIsFallback(url string, timeout time.Duration) (map[string]interface{}, []byte, *goose.Article, error) {
	var s string

	snapshots, err := archiveis.Search(url, timeout)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(snapshots) > 0 {
		content, err := download(url, timeout)
		if err != nil {
			return nil, nil, nil, err
		}
		s = string(content)
	}
	//s, err := archiveis.Capture(url)
	//if err != nil {
	//	return nil, nil, nil, err
	//}
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

func handlePDF(url string) ([]byte, error) {
	var (
		ch   = make(chan error, 1)
		text []byte
		cmd  = exec.Command("bash", "-c", fmt.Sprintf("set -o errexit && set -o pipefail && set -o nounset && curl -k -sSL -o /tmp/pdf.pdf %q | pdf2htmlEX --auto-hint 1 --correct-text-visibility 1 --process-annotation 1 /tmp/pdf.pdf /tmp/pdf.html", url))
	)

	go func() {
		out, err := cmd.CombinedOutput()
		if err != nil {
			ch <- fmt.Errorf("converting PDF to HTML: %s (output=%v)", err, string(out))
			return
		}
		if text, err = ioutil.ReadFile("/tmp/pdf.html"); err != nil {
			ch <- err
			return
		}
		ch <- nil
	}()

	select {
	case err := <-ch:
		if err != nil {
			return nil, err
		}

	case <-time.After(PDFProcessorTimeout):
		log.Errorf("Timed out after %s processing PDF from %v", PDFProcessorTimeout, url)
		if err := cmd.Process.Kill(); err != nil {
			log.Errorf("Failed to kill PDF converter process: %s", err)
			return nil, fmt.Errorf("killing PDF converter process: %s", err)
		}
		return nil, fmt.Errorf("timed out after %s processing PDF from %v", PDFProcessorTimeout, url)
	}

	return text, nil
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
		fmt.Sprintf(": && set -o errexit && cd %q && source ../venv/bin/activate && python ../nlpweb.py %v", oslib.PathDirName(os.Args[0]), NLPWebAddr),
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

func download(url string, timeout time.Duration) ([]byte, error) {
	req, err := newGetRequest(url)
	if err != nil {
		return nil, err
	}

	client := newClient(timeout)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		log.WithField("url", url).WithField("status-code", resp.StatusCode).Error("Received non-2xx response from URL (falling back to archive.is search)")
		// Fallback to archive.is.
		snapshots, err := archiveis.Search(url, timeout)
		log.WithField("url", url).WithField("snapshots", len(snapshots)).Info("Found archive.is snapshots")
		if err == nil && len(snapshots) > 0 {
			if req, err = newGetRequest(snapshots[0].URL); err == nil {
				if resp, err = client.Do(req); err != nil {
					log.WithField("url", snapshots[0].URL).Errorf("Received error from URL: %s", err)
					return nil, fmt.Errorf("even archive.is fallback failed: %s", err)
				}
				if resp.StatusCode/100 != 2 {
					log.WithField("url", snapshots[0].URL).WithField("status-code", resp.StatusCode).Error("Received non-2xx response from URL")
					return nil, fmt.Errorf("even archive.is fallback produced non-2xx response status code=%v", resp.StatusCode)
				}
			}
		}
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body from %v: %s", url, err)
	}
	if err := resp.Body.Close(); err != nil {
		return data, fmt.Errorf("closing body from %v: %s", url, err)
	}

	return data, nil

}

func newGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get request to %v: %s", url, err)
	}

	split := strings.Split(url, "://")
	proto := split[0]
	hostname := strings.Split(split[1], "/")[0]

	req.Header.Set("Host", hostname)
	req.Header.Set("Origin", hostname)
	req.Header.Set("Authority", hostname)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Referer", fmt.Sprintf("%v://%v", proto, hostname))

	return req, nil
}

func newClient(timeout time.Duration) *http.Client {
	c := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: timeout,
			}).Dial,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return c
}
