// Package export writes the official 3-sheet upload file
// (column set fixed by the real decrypted sheet in data/).
package export

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

func Export(g *model.Generated, outPath string) error {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "campaigns")
	if _, err := f.NewSheet("adgroups"); err != nil {
		return err
	}
	if _, err := f.NewSheet("ads"); err != nil {
		return err
	}

	dateFmt := "yyyy-mm-dd"
	dateStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &dateFmt})
	if err != nil {
		return err
	}

	setRow(f, "campaigns", 1, []any{"campaign_name", "budget_max", "budget_type",
		"launch_date", "end_date", "objective", "target_countries"})
	for i, c := range g.Campaigns {
		row := i + 2
		setRow(f, "campaigns", row, []any{c.CampaignName, c.BudgetMax, c.BudgetType,
			nil, nil, c.Objective, jsonArr(c.TargetCountries)})
		if err := setDate(f, "campaigns", "D", row, c.LaunchDate, dateStyle); err != nil {
			return err
		}
		if err := setDate(f, "campaigns", "E", row, c.EndDate, dateStyle); err != nil {
			return err
		}
	}

	setRow(f, "adgroups", 1, []any{"campaign_name", "adgroup_name", "max_bid", "keywords"})
	for i, ag := range g.Adgroups {
		texts := make([]string, len(ag.Keywords))
		for j, k := range ag.Keywords {
			texts[j] = k.Text
		}
		// max_bid (column C) is intentionally never written: filling it breaks upload.
		setRow(f, "adgroups", i+2, []any{ag.CampaignName, ag.AdgroupName, nil, jsonArr(texts)})
	}

	setRow(f, "ads", 1, []any{"adgroup_name", "title", "copy", "link", "image_link"})
	for i, ad := range g.Ads {
		setRow(f, "ads", i+2, []any{ad.AdgroupName, ad.Title, ad.Copy, ad.Link, ad.ImageLink})
	}
	return f.SaveAs(outPath)
}

func setDate(f *excelize.File, sheet, col string, row int, ymd string, style int) error {
	t, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		return fmt.Errorf("%s!%s%d: 날짜 형식 오류 %q (YYYY-MM-DD)", sheet, col, row, ymd)
	}
	cell := fmt.Sprintf("%s%d", col, row)
	if err := f.SetCellValue(sheet, cell, t); err != nil {
		return err
	}
	return f.SetCellStyle(sheet, cell, cell, style)
}

func setRow(f *excelize.File, sheet string, row int, vals []any) {
	for i, v := range vals {
		if v == nil {
			continue
		}
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, v)
	}
}

func jsonArr(ss []string) string {
	b, _ := json.Marshal(ss)
	return string(b)
}
