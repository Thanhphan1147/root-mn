// Command rmn parses, validates, replays, and serves Root Machine
// Notation (RMN) v3.0 game logs for the board game ROOT.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/Thanhphan1147/root-mn/internal/rmn"
)

const usage = `rmn - Root Machine Notation (RMN) v3.0 tool

Usage:
  rmn parse     <file.rmn>            Parse and print the JSON envelope
  rmn validate  <file.rmn>            Parse + replay; report errors/warnings
  rmn replay    <file.rmn>            Print the final computed state + hash
  rmn canonical <file.rmn>            Print canonical text form
  rmn apply     --state <state.json> --action "<event line>"
  rmn serve     [--addr :8080]        Start the web playtest/analysis tool

Flags:
  --pretty   pretty-print JSON (default true)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "parse":
		cmdParse(os.Args[2:])
	case "validate":
		cmdValidate(os.Args[2:])
	case "replay":
		cmdReplay(os.Args[2:])
	case "canonical":
		cmdCanonical(os.Args[2:])
	case "apply":
		cmdApply(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return string(b)
}

func emit(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

func cmdParse(args []string) {
	fs := flag.NewFlagSet("parse", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	log := rmn.ParseLog(readFile(fs.Arg(0)))
	b, err := log.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
	if len(log.Errors) > 0 {
		os.Exit(1)
	}
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	log := rmn.ParseLog(readFile(fs.Arg(0)))
	res := rmn.Replay(log)
	report := struct {
		OK       bool     `json:"ok"`
		Events   int      `json:"events"`
		Hash     string   `json:"hash"`
		Errors   []string `json:"errors,omitempty"`
		Warnings []string `json:"warnings,omitempty"`
	}{OK: len(res.Errors) == 0, Events: res.Events, Hash: res.Hash, Errors: res.Errors, Warnings: res.Warnings}
	emit(report)
	if !report.OK {
		os.Exit(1)
	}
}

func cmdReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	log := rmn.ParseLog(readFile(fs.Arg(0)))
	res := rmn.Replay(log)
	emit(res)
	if len(res.Errors) > 0 {
		os.Exit(1)
	}
}

func cmdCanonical(args []string) {
	fs := flag.NewFlagSet("canonical", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	log := rmn.ParseLog(readFile(fs.Arg(0)))
	fmt.Print(log.CanonicalText())
}

func cmdApply(args []string) {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	statePath := fs.String("state", "", "path to a JSON state")
	action := fs.String("action", "", "RMN event line")
	fs.Parse(args)
	if *statePath == "" || *action == "" {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var st rmn.State
	if err := json.Unmarshal([]byte(readFile(*statePath)), &st); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid state: %v\n", err)
		os.Exit(1)
	}
	ev, perr := rmn.ParseEventLine(*action, 0)
	if perr != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", perr)
		os.Exit(1)
	}
	fr := rmn.Fold(&st, ev)
	emit(map[string]any{"event": ev, "state": &st, "errors": fr.Errors, "warnings": fr.Warnings})
}
