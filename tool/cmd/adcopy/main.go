package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"adcopy/internal/export"
	"adcopy/internal/merge"
	"adcopy/internal/model"
	"adcopy/internal/review"
	"adcopy/internal/urlcheck"
	"adcopy/internal/validate"
	"adcopy/internal/workbook"
)

// Keep in sync with .claude-plugin/plugin.json "version".
const version = "0.5.1"

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
	case "merge":
		// adcopy merge -o <merged.json> <chunk.json>...
		if len(os.Args) < 5 || os.Args[2] != "-o" {
			usage()
		}
		g, err := merge.Merge(os.Args[4:])
		if err != nil {
			fail(err)
		}
		f, err := os.Create(os.Args[3])
		if err != nil {
			fail(err)
		}
		enc := json.NewEncoder(f)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(g); err != nil {
			f.Close()
			fail(err)
		}
		if err := f.Close(); err != nil {
			fail(err)
		}
		writeJSON(map[string]any{"written": os.Args[3],
			"campaigns": len(g.Campaigns), "adgroups": len(g.Adgroups), "ads": len(g.Ads)})
	case "review-out":
		// adcopy review-out <generated.json> -o <review.xlsx> [--report <validate-report.json>]
		if (len(os.Args) != 5 && len(os.Args) != 7) || os.Args[3] != "-o" {
			usage()
		}
		g, err := model.Load(os.Args[2])
		if err != nil {
			fail(err)
		}
		var rep *validate.Report
		if len(os.Args) == 7 {
			if os.Args[5] != "--report" {
				usage()
			}
			b, err := os.ReadFile(os.Args[6])
			if err != nil {
				fail(err)
			}
			rep = &validate.Report{}
			if err := json.Unmarshal(b, rep); err != nil {
				fail(fmt.Errorf("parse %s: %w", os.Args[6], err))
			}
		}
		if err := review.WriteReview(g, rep, os.Args[4]); err != nil {
			fail(err)
		}
		writeJSON(map[string]any{"written": os.Args[4], "adgroups": len(g.Adgroups), "ads": len(g.Ads)})
	case "review-in":
		// adcopy review-in <review.xlsx> <generated.json>
		if len(os.Args) != 4 {
			usage()
		}
		g, err := model.Load(os.Args[3])
		if err != nil {
			fail(err)
		}
		res, err := review.ReadReview(os.Args[2], g)
		if err != nil {
			fail(err)
		}
		writeJSON(res)
		if len(res.Problems) > 0 {
			os.Exit(1)
		}
	case "export":
		// adcopy export <approved.json> -o <final.xlsx>
		if len(os.Args) != 5 || os.Args[3] != "-o" {
			usage()
		}
		g, err := model.Load(os.Args[2])
		if err != nil {
			fail(err)
		}
		rep := validate.Validate(g)
		if !rep.OK {
			writeJSON(rep)
			fmt.Fprintln(os.Stderr, "adcopy: 검증 오류가 있어 export를 중단합니다")
			os.Exit(1)
		}
		if err := export.Export(g, os.Args[4]); err != nil {
			fail(err)
		}
		writeJSON(map[string]any{"written": os.Args[4],
			"campaigns": len(g.Campaigns), "adgroups": len(g.Adgroups), "ads": len(g.Ads)})
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
  merge -o <merged.json> <chunk.json>...  merge generation chunk files
  review-out <generated.json> -o <xlsx> [--report <report.json>]
                                          write review workbook (report: merge
                                          per-row automatic validation findings)
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
