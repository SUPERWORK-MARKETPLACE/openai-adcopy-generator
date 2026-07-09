// Package review writes and reads the operator review workbook (엑셀 왕복).
//
// Column roles (개발요청서 6-11 근거 추적):
//   - validation_status: machine/AI 자동 검수 결과 — 형식·정책·사실성·연결성
//     검사 결과와 문제 유형을 표시한다. 운영자는 읽기만 한다.
//   - 검수상태: 운영자 판정 드롭다운 (무수정 승인 등 5종). review-in이 읽는다.
//   - review_comment: 운영자·광고주 수정 의견·재생성 사유 기입란.
package review

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
	"adcopy/internal/validate"
)

// StatusColumnHeader is the operator-decision dropdown column.
const StatusColumnHeader = "검수상태"

var adgroupHeaders = []string{
	"campaign_name", "adgroup_name", "keywords", "keywords_origin",
	"validation_status", StatusColumnHeader, "review_comment",
	"source_type", "source_url", "source_excerpt", "generation_basis",
	"confidence_score", "exclusion_reason",
}

var adHeaders = []string{
	"ad_name", "adgroup_name", "title", "title_글자수", "copy", "copy_글자수",
	"link", "image_link", "validation_status", StatusColumnHeader, "review_comment",
	"source_type", "source_url", "source_excerpt", "generation_basis",
	"confidence_score", "exclusion_reason",
}

// WriteReview writes the review workbook. rep may be nil (no per-row merge of
// automatic validation findings).
func WriteReview(g *model.Generated, rep *validate.Report, outPath string) error {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "요약")
	if _, err := f.NewSheet("adgroups_검수"); err != nil {
		return err
	}
	if _, err := f.NewSheet("ads_검수"); err != nil {
		return err
	}

	writeSummary(f, g)
	findings := findingsByID(rep)

	setRow(f, "adgroups_검수", 1, toAny(adgroupHeaders))
	for i, ag := range g.Adgroups {
		texts := make([]string, len(ag.Keywords))
		origins := make([]string, len(ag.Keywords))
		for j, k := range ag.Keywords {
			texts[j], origins[j] = k.Text, k.Origin
		}
		setRow(f, "adgroups_검수", i+2, []any{
			ag.CampaignName, ag.AdgroupName, jsonArr(texts), jsonArr(origins),
			validationCell(ag.Trace.ValidationStatus, findings["adgroups\x00"+ag.AdgroupName]),
			"", ag.Trace.ReviewComment,
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
			validationCell(ad.Trace.ValidationStatus, findings["ads\x00"+ad.AdName]),
			"", ad.Trace.ReviewComment,
			ad.Trace.SourceType, ad.Trace.SourceURL, ad.Trace.SourceExcerpt,
			ad.Trace.GenerationBasis, ad.Trace.ConfidenceScore, ad.Trace.ExclusionReason,
		})
	}

	if err := addStatusDropdown(f, "adgroups_검수", "F", len(g.Adgroups)); err != nil {
		return err
	}
	if err := addStatusDropdown(f, "ads_검수", "J", len(g.Ads)); err != nil {
		return err
	}

	f.SetColWidth("adgroups_검수", "C", "D", 60)
	f.SetColWidth("adgroups_검수", "E", "E", 40)
	f.SetColWidth("ads_검수", "C", "F", 40)
	f.SetColWidth("ads_검수", "I", "I", 40)
	return f.SaveAs(outPath)
}

// findingsByID groups validation report findings per entity row.
// Key format: entity + "\x00" + id.
func findingsByID(rep *validate.Report) map[string][]string {
	m := map[string][]string{}
	if rep == nil {
		return m
	}
	add := func(fs []validate.Finding, kind string) {
		for _, fd := range fs {
			k := fd.Entity + "\x00" + fd.ID
			m[k] = append(m[k], fmt.Sprintf("%s(%s): %s", kind, fd.Rule, fd.Message))
		}
	}
	add(rep.Errors, "오류")
	add(rep.Warnings, "경고")
	return m
}

// validationCell merges the AI-written trace status/notes with the machine
// findings for one row. Empty everything → "통과".
func validationCell(traceStatus string, findings []string) string {
	parts := []string{}
	if s := strings.TrimSpace(traceStatus); s != "" {
		parts = append(parts, s)
	}
	parts = append(parts, findings...)
	if len(parts) == 0 {
		return "통과"
	}
	return strings.Join(parts, " | ")
}

func writeSummary(f *excelize.File, g *model.Generated) {
	needs, excluded := 0, 0
	for _, ad := range g.Ads {
		if strings.Contains(ad.Trace.ValidationStatus, model.StatusNeedsAdvertiser) {
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
		{"안내", "validation_status 열은 자동 검수 결과(읽기 전용)입니다. " +
			StatusColumnHeader + " 열의 드롭다운에서 판정을 선택하고 review_comment에 의견을 남겨 주세요."},
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
