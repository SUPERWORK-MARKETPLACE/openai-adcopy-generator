package review

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

type AdgroupReview struct {
	AdgroupName string `json:"adgroup_name"`
	// ReviewStatus: operator decision from the 검수상태 dropdown column.
	ReviewStatus  string   `json:"review_status"`
	ReviewComment string   `json:"review_comment"`
	Keywords      []string `json:"keywords"`
}

type AdReview struct {
	AdName        string `json:"ad_name"`
	AdgroupName   string `json:"adgroup_name"`
	Title         string `json:"title"`
	Copy          string `json:"copy"`
	ReviewStatus  string `json:"review_status"`
	ReviewComment string `json:"review_comment"`
}

type ReviewResult struct {
	Adgroups []AdgroupReview `json:"adgroups"`
	Ads      []AdReview      `json:"ads"`
	Problems []string        `json:"problems"`
	// FlaggedApproved lists rows whose validation_status carries an automatic
	// check flag but were nevertheless marked 무수정 승인 (R6). Surfaced so the
	// operator/advertiser can confirm — never blocks the pipeline.
	FlaggedApproved []string `json:"flagged_approved"`
}

func ReadReview(reviewPath string, g *model.Generated) (*ReviewResult, error) {
	f, err := excelize.OpenFile(reviewPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", reviewPath, err)
	}
	defer f.Close()

	res := &ReviewResult{Problems: []string{}, FlaggedApproved: []string{}}
	knownAds := map[string]bool{}
	for _, ad := range g.Ads {
		knownAds[ad.AdName] = true
	}
	knownAdgroups := map[string]bool{}
	for _, ag := range g.Adgroups {
		knownAdgroups[ag.AdgroupName] = true
	}
	validStatus := map[string]bool{"": true}
	for _, s := range model.AllStatuses {
		validStatus[s] = true
	}

	agRows, agIdx, err := sheetByHeader(f, "adgroups_검수")
	if err != nil {
		return nil, err
	}
	for rn, row := range agRows {
		if allCellsEmpty(row) {
			continue
		}
		get := func(h string) string { return cellAt(row, agIdx, h) }
		ar := AdgroupReview{
			AdgroupName:   get("adgroup_name"),
			ReviewStatus:  get(StatusColumnHeader),
			ReviewComment: get("review_comment"),
		}
		if !knownAdgroups[ar.AdgroupName] {
			res.Problems = append(res.Problems, fmt.Sprintf("adgroups_검수 %d행: 원본에 없는 adgroup_name %q", rn+2, ar.AdgroupName))
		}
		if !validStatus[ar.ReviewStatus] {
			res.Problems = append(res.Problems, fmt.Sprintf("adgroups_검수 %d행: 허용되지 않는 검수상태 %q", rn+2, ar.ReviewStatus))
		}
		if kwRaw := get("keywords"); strings.TrimSpace(kwRaw) != "" {
			if err := json.Unmarshal([]byte(kwRaw), &ar.Keywords); err != nil {
				res.Problems = append(res.Problems, fmt.Sprintf("adgroups_검수 %d행: keywords JSON 파싱 실패", rn+2))
			}
		}
		if ar.ReviewStatus == model.StatusApproved && model.NeedsAdvertiserDefault(get("validation_status")) {
			res.FlaggedApproved = append(res.FlaggedApproved, "adgroups:"+ar.AdgroupName)
		}
		res.Adgroups = append(res.Adgroups, ar)
	}

	adRows, adIdx, err := sheetByHeader(f, "ads_검수")
	if err != nil {
		return nil, err
	}
	for rn, row := range adRows {
		if allCellsEmpty(row) {
			continue
		}
		get := func(h string) string { return cellAt(row, adIdx, h) }
		ar := AdReview{
			AdName:        get("ad_name"),
			AdgroupName:   get("adgroup_name"),
			Title:         get("title"),
			Copy:          get("copy"),
			ReviewStatus:  get(StatusColumnHeader),
			ReviewComment: get("review_comment"),
		}
		if !knownAds[ar.AdName] {
			res.Problems = append(res.Problems, fmt.Sprintf("ads_검수 %d행: 원본에 없는 ad_name %q", rn+2, ar.AdName))
		}
		if !validStatus[ar.ReviewStatus] {
			res.Problems = append(res.Problems, fmt.Sprintf("ads_검수 %d행: 허용되지 않는 검수상태 %q", rn+2, ar.ReviewStatus))
		}
		if ar.ReviewStatus == model.StatusApproved && model.NeedsAdvertiserDefault(get("validation_status")) {
			res.FlaggedApproved = append(res.FlaggedApproved, "ads:"+ar.AdName)
		}
		res.Ads = append(res.Ads, ar)
	}
	return res, nil
}

// sheetByHeader returns data rows and a header→column-index map.
func sheetByHeader(f *excelize.File, sheet string) ([][]string, map[string]int, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("sheet %s: %w", sheet, err)
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("sheet %s: 비어 있음", sheet)
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[h] = i
	}
	return rows[1:], idx, nil
}

func cellAt(row []string, idx map[string]int, header string) string {
	i, ok := idx[header]
	if !ok || i >= len(row) {
		return ""
	}
	return row[i]
}

// allCellsEmpty reports whether every cell in row is blank. GetRows can surface
// such a row when an operator clears a row's contents in Excel without deleting
// the row itself (see TestReadReviewSkipsClearedRows).
func allCellsEmpty(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
