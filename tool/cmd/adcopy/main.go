package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "version":
		writeJSON(map[string]string{"name": "adcopy", "version": version})
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: adcopy <command> [args]
commands:
  version                                 print version
  inspect <in.xlsx>                       dump advertiser workbook to JSON
  checkurls <urls.json>                   check URL accessibility
  validate <generated.json>               run format validation rules
  review-out <generated.json> -o <xlsx>   write review workbook
  review-in <review.xlsx> <generated.json> read back review workbook
  export <approved.json> -o <xlsx>        write official 3-sheet upload file`)
	os.Exit(2)
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "adcopy:", err)
	os.Exit(2)
}
