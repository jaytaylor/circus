package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"

	"github.com/gigawattio/oslib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	OutputFormat string
	Quiet        bool
	Verbose      bool
)

func init() {
	rootCmd.AddCommand(versionCmd)
	// rootCmd.PersistentFlags().StringVarP(&OutputFormat, "output", "o", "text", `Output format, one of "json", "html", "text", "yaml"`)
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Activate quiet log output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Activate verbose log output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		errorExit(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "wrap",
	Short: "wrap it",
	Long:  "wrap it up",
	// Args:  cobra.MinimumNArgs(1),
	PreRun: func(_ *cobra.Command, _ []string) {
		initLogging()
	},
	Run: func(cmd *cobra.Command, args []string) {

		// stdout := &bytes.Buffer{}
		// stderr := &bytes.Buffer{}

		// TODO: Consider using uWSGI?  Would that be useful?  Maybe, not.
		nlpCmd := exec.Command("/usr/bin/env", "bash", "-c",
			fmt.Sprintf(": && set -o errexit && cd %q && source venv/bin/activate && python nlpweb.py", oslib.PathDirName(os.Args[0])),
		)

		{
			r, w := io.Pipe()
			nlpCmd.Stdout = w
			go func() {
				scanner := bufio.NewScanner(r)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					log.Debugf("[nlpweb][stdout] %v", scanner.Text())
					// fmt.Printf("[nlpweb][stdout] %v\n", scanner.Text())
				}
			}()
		}

		{
			r, w := io.Pipe()
			nlpCmd.Stderr = w
			go func() {
				scanner := bufio.NewScanner(r)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					log.Infof("[nlpweb][stderr] %v", scanner.Text())
					// fmt.Printf("[nlpweb][stderr] %v\n", scanner.Text())
				}
			}()
		}

		// nlpCmd.Stdout = stdout
		// nlpCmd.Stderr = stderr

		if err := nlpCmd.Start(); err != nil {
			errorExit(err)
		}

		log.Debugf("Started nlpweb, pid=%v", nlpCmd.Process.Pid)

		// go func() {
		// 	for {
		// 		time.Sleep(1 * time.Second)
		// 		log.Debugf("%# v", nlpCmd.ProcessState)
		// 	}
		// }()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		signal.Notify(sig, os.Kill)

		<-sig // Wait for ^C signal.
		fmt.Fprintln(os.Stderr, "\nInterrupt or kill signal detected, shutting down..")

		if err := nlpCmd.Process.Kill(); err != nil {
			errorExit(err)
		}
		log.Info("Killed nlpweb") // out=%v err=%v", stdout.String(), stderr.String())
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
