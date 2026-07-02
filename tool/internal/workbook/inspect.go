// Package workbook reads advertiser-supplied xlsx files generically.
package workbook

import (
	"bytes"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

type WorkbookDump struct {
	File   string      `json:"file"`
	Sheets []SheetDump `json:"sheets"`
}

type SheetDump struct {
	Name    string              `json:"name"`
	Headers []string            `json:"headers"`
	Rows    []map[string]string `json:"rows"`
}

func Inspect(path string) (*WorkbookDump, error) {
	if err := checkMagic(path); err != nil {
		return nil, err
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	dump := &WorkbookDump{File: path}
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			return nil, fmt.Errorf("sheet %s: %w", name, err)
		}
		sd := SheetDump{Name: name}
		for ri, row := range rows {
			if allEmpty(row) {
				continue
			}
			if sd.Headers == nil {
				sd.Headers = headerNames(row)
				continue
			}
			m := map[string]string{}
			for ci, h := range sd.Headers {
				if ci < len(row) {
					m[h] = row[ci]
				} else {
					m[h] = ""
				}
			}
			_ = ri
			sd.Rows = append(sd.Rows, m)
		}
		dump.Sheets = append(dump.Sheets, sd)
	}
	return dump, nil
}

// checkMagic distinguishes standard OOXML (PK) from Korean DRM containers
// (SCDSA) and legacy/encrypted CFB files (D0 CF), for clear operator errors.
func checkMagic(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	switch {
	case len(b) >= 2 && bytes.HasPrefix(b, []byte("PK")):
		return nil
	case bytes.HasPrefix(b, []byte("SCDSA")):
		return fmt.Errorf("%s: DRM으로 암호화된 파일입니다. 표준 xlsx로 다시 저장(복호화)한 파일을 사용하세요", path)
	case len(b) >= 2 && b[0] == 0xD0 && b[1] == 0xCF:
		return fmt.Errorf("%s: 암호화되었거나 구형(xls) 형식입니다. 암호 없는 표준 xlsx로 다시 저장하세요", path)
	default:
		return fmt.Errorf("%s: xlsx(ZIP) 형식이 아닙니다", path)
	}
}

func allEmpty(row []string) bool {
	for _, c := range row {
		if c != "" {
			return false
		}
	}
	return true
}

// headerNames fills blank header cells with the Excel column name (A, B, ...).
func headerNames(row []string) []string {
	out := make([]string, len(row))
	for i, h := range row {
		if h == "" {
			name, _ := excelize.ColumnNumberToName(i + 1)
			out[i] = name
		} else {
			out[i] = h
		}
	}
	return out
}
