// Package review writes and reads the operator review workbook (엑셀 왕복).
package review

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

var adgroupHeaders = []string{
	"campaign_name", "adgroup_name", "keywords", "keywords_origin",
	"validation_status", "review_comment",
	"source_type", "source_url", "source_excerpt", "generation_basis",
	"confidence_score", "exclusion_reason",
}

var adHeaders = []string{
	"ad_name", "adgroup_name", "title", "title_글자수", "copy", "copy_글자수",
	"link", "image_link", "validation_status", "review_comment",
	"source_type", "source_url", "source_excerpt", "generation_basis",
	"confidence_score", "exclusion_reason",
}

func WriteReview(g *model.Generated, outPath string) error {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "요약")
	if _, err := f.NewSheet("adgroups_검수"); err != nil {
		return err
	}
	if _, err := f.NewSheet("ads_검수"); err != nil {
		return err
	}

	writeSummary(f, g)

	setRow(f, "adgroups_검수", 1, toAny(adgroupHeaders))
	for i, ag := range g.Adgroups {
		texts := make([]string, len(ag.Keywords))
		origins := make([]string, len(ag.Keywords))
		for j, k := range ag.Keywords {
			texts[j], origins[j] = k.Text, k.Origin
		}
		setRow(f, "adgroups_검수", i+2, []any{
			ag.CampaignName, ag.AdgroupName, jsonArr(texts), jsonArr(origins),
			ag.Trace.ValidationStatus, ag.Trace.ReviewComment,
			ag.Trace.SourceType, ag.Trace.SourceURL, ag.Trace.SourceExcerpt,
			ag.Trace.GenerationBasis, ag.Trace.ConfidenceScore, ag.Trace.ExclusionReason,
		})
	}

	setRow(f, "ads_검수", 1, toAny(adHeaders))
	for i, ad := range g.Ads {
		setRow(f, "ads_검수", i+2, []any{
			ad.AdName, ad.AdgroupName,
			ad.Title, utf8.RuneCountInString(strings.TrimSpace(ad.Title)),
			ad.Copy, utf8.RuneCountInString(strings.TrimSpace(ad.Copy)),
			ad.Link, ad.ImageLink,
			ad.Trace.ValidationStatus, ad.Trace.ReviewComment,
			ad.Trace.SourceType, ad.Trace.SourceURL, ad.Trace.SourceExcerpt,
			ad.Trace.GenerationBasis, ad.Trace.ConfidenceScore, ad.Trace.ExclusionReason,
		})
	}

	if err := addStatusDropdown(f, "adgroups_검수", "E", len(g.Adgroups)); err != nil {
		return err
	}
	if err := addStatusDropdown(f, "ads_검수", "I", len(g.Ads)); err != nil {
		return err
	}

	f.SetColWidth("adgroups_검수", "C", "D", 60)
	f.SetColWidth("ads_검수", "C", "F", 40)
	return f.SaveAs(outPath)
}

func writeSummary(f *excelize.File, g *model.Generated) {
	needs, excluded := 0, 0
	for _, ad := range g.Ads {
		if ad.Trace.ValidationStatus == model.StatusNeedsAdvertiser {
			needs++
		}
		if ad.Trace.ExclusionReason != "" {
			excluded++
		}
	}
	rows := [][]any{
		{"항목", "값"},
		{"캠페인 수", len(g.Campaigns)},
		{"광고그룹 수", len(g.Adgroups)},
		{"광고 수", len(g.Ads)},
		{"광고주 확인 필요(광고)", needs},
		{"제외 사유 있는 광고", excluded},
		{"안내", "validation_status 열의 드롭다운에서 상태를 선택하고 review_comment에 의견을 남겨 주세요."},
		{"상태값", strings.Join(model.AllStatuses, " / ")},
	}
	for i, r := range rows {
		setRow(f, "요약", i+1, r)
	}
	f.SetColWidth("요약", "A", "A", 24)
	f.SetColWidth("요약", "B", "B", 80)
}

func addStatusDropdown(f *excelize.File, sheet, col string, n int) error {
	if n == 0 {
		return nil
	}
	dv := excelize.NewDataValidation(true)
	dv.Sqref = fmt.Sprintf("%s2:%s%d", col, col, n+1)
	if err := dv.SetDropList(model.AllStatuses); err != nil {
		return err
	}
	return f.AddDataValidation(sheet, dv)
}

func setRow(f *excelize.File, sheet string, row int, vals []any) {
	for i, v := range vals {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, v)
	}
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func jsonArr(ss []string) string {
	b, _ := json.Marshal(ss)
	return string(b)
}
