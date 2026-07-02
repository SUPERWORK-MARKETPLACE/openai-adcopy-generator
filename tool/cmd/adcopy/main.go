package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"adcopy/internal/model"
	"adcopy/internal/review"
	"adcopy/internal/urlcheck"
	"adcopy/internal/validate"
	"adcopy/internal/workbook"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "version":
		writeJSON(map[string]string{"name": "adcopy", "version": version})
	case "inspect":
		if len(os.Args) != 3 {
			usage()
		}
		d, err := workbook.Inspect(os.Args[2])
		if err != nil {
			fail(err)
		}
		writeJSON(d)
	case "checkurls":
		if len(os.Args) != 3 {
			usage()
		}
		b, err := os.ReadFile(os.Args[2])
		if err != nil {
			fail(err)
		}
		var in struct {
			URLs []string `json:"urls"`
		}
		if err := json.Unmarshal(b, &in); err != nil {
			fail(fmt.Errorf("parse %s: %w", os.Args[2], err))
		}
		results := urlcheck.Check(in.URLs, 10*time.Second)
		allOK := true
		for _, r := range results {
			if !r.OK {
				allOK = false
			}
		}
		writeJSON(map[string]any{"ok": allOK, "results": results})
		if !allOK {
			os.Exit(1)
		}
	case "validate":
		if len(os.Args) != 3 {
			usage()
		}
		g, err := model.Load(os.Args[2])
		if err != nil {
			fail(err)
		}
		rep := validate.Validate(g)
		writeJSON(rep)
		if !rep.OK {
			os.Exit(1)
		}
	case "review-out":
		// adcopy review-out <generated.json> -o <review.xlsx>
		if len(os.Args) != 5 || os.Args[3] != "-o" {
			usage()
		}
		g, err := model.Load(os.Args[2])
		if err != nil {
			fail(err)
		}
		if err := review.WriteReview(g, os.Args[4]); err != nil {
			fail(err)
		}
		writeJSON(map[string]any{"written": os.Args[4], "adgroups": len(g.Adgroups), "ads": len(g.Ads)})
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
