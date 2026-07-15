package review

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

// writeEditedReview simulates the operator: fills statuses, edits one copy.
func writeEditedReview(t *testing.T, g *model.Generated, edit func(f *excelize.File)) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "review.xlsx")
	if err := WriteReview(g, nil, p); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(p)
	if err != nil {
		t.Fatal(err)
	}
	edit(f)
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return p
}

func TestReadReviewRoundTrip(t *testing.T) {
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {
		f.SetCellValue("ads_검수", "J2", model.StatusApprovedEdited) // 검수상태
		f.SetCellValue("ads_검수", "E2", "수정된 카피 문구로 교체합니다")   // copy
		f.SetCellValue("ads_검수", "K2", "혜택 표현 수정")             // review_comment
		f.SetCellValue("adgroups_검수", "F2", model.StatusApproved) // 검수상태
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", res.Problems)
	}
	ad := res.Ads[0]
	if ad.ReviewStatus != model.StatusApprovedEdited || ad.Copy != "수정된 카피 문구로 교체합니다" {
		t.Fatalf("edited ad not read back: %+v", ad)
	}
	if ad.ReviewComment != "혜택 표현 수정" {
		t.Fatalf("review_comment not read back: %+v", ad)
	}
	if res.Adgroups[0].ReviewStatus != model.StatusApproved {
		t.Fatalf("adgroup status not read back: %+v", res.Adgroups[0])
	}
	if len(res.Adgroups[0].Keywords) != 5 {
		t.Fatalf("keywords not parsed: %+v", res.Adgroups[0].Keywords)
	}
	// 수정 후 승인은 flagged_approved 대상이 아니다(무수정 승인만 해당).
	if len(res.FlaggedApproved) != 0 {
		t.Fatalf("unexpected flagged_approved: %v", res.FlaggedApproved)
	}
}

func TestReadReviewFlagsBulkApprovedRows(t *testing.T) {
	// R6: 플래그 행(validation_status에 트리거 포함)이 무수정 승인으로 덮이면
	// flagged_approved로 표면화한다 — 차단은 하지 않는다.
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {
		f.SetCellValue("ads_검수", "J2", model.StatusApproved)      // 플래그 행 일괄 승인 시뮬레이션
		f.SetCellValue("adgroups_검수", "F2", model.StatusApproved) // 깨끗한 행(통과) 승인
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.FlaggedApproved) != 1 || res.FlaggedApproved[0] != "ads:KID_01_001" {
		t.Fatalf("want flagged_approved [ads:KID_01_001], got %v", res.FlaggedApproved)
	}
}

func TestReadReviewFlagsProblems(t *testing.T) {
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {
		f.SetCellValue("ads_검수", "A2", "UNKNOWN_AD")     // ad_name not in generated
		f.SetCellValue("ads_검수", "J2", "승인함")           // invalid 검수상태 value
		f.SetCellValue("adgroups_검수", "C2", "[broken json") // keywords JSON 깨짐
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) < 3 {
		t.Fatalf("want >=3 problems (unknown ad, bad status, bad keywords), got %v", res.Problems)
	}
}

// 0715 요청 1: 검수상태 blank default — a flagged row left unreviewed must be
// surfaced as flagged_unreviewed so finalize's 미검수 report keeps the flag
// context. Clean unreviewed rows (adgroup here) stay out.
func TestReadReviewFlagsUnreviewedFlaggedRows(t *testing.T) {
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {}) // nothing filled in
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.FlaggedUnreviewed) != 1 || res.FlaggedUnreviewed[0] != "ads:KID_01_001" {
		t.Fatalf("want flagged_unreviewed [ads:KID_01_001], got %v", res.FlaggedUnreviewed)
	}
	if len(res.FlaggedApproved) != 0 {
		t.Fatalf("unexpected flagged_approved: %v", res.FlaggedApproved)
	}
}

// 0715 요청 1: 광고주 확인 필요 is no longer a 검수상태 value — entering it in
// the dropdown column must surface as a problem (validation_status 전용 플래그).
func TestReadReviewRejectsNeedsAdvertiserAsStatus(t *testing.T) {
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {
		f.SetCellValue("ads_검수", "J2", model.StatusNeedsAdvertiser)
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 1 || !strings.Contains(res.Problems[0], model.StatusNeedsAdvertiser) {
		t.Fatalf("want 1 problem rejecting %q as 검수상태, got %v", model.StatusNeedsAdvertiser, res.Problems)
	}
}

// TestReadReviewSkipsClearedRows reproduces an operator clearing a row's cells
// in Excel without deleting the row. excelize's GetRows only drops an all-empty
// row when a later row in the sheet still has content (it pads the gap with a
// nil row instead of trimming it) — a cleared row with nothing below it is
// trimmed away on its own and never reaches ReadReview. So this test moves the
// real ad row down to row 3 and clears row 2 in its place, which reliably
// reproduces the phantom nil row ReadReview must skip.
func TestReadReviewSkipsClearedRows(t *testing.T) {
	g := sampleGenerated()
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q"}
	p := writeEditedReview(t, g, func(f *excelize.File) {
		for _, col := range cols {
			v, _ := f.GetCellValue("ads_검수", col+"2")
			f.SetCellValue("ads_검수", col+"3", v)
		}
		f.SetCellValue("ads_검수", "J3", model.StatusApproved) // 검수상태 on the moved real row
		for _, col := range cols {
			f.SetCellValue("ads_검수", col+"2", "") // clear row 2 entirely (phantom row, now before the real row)
		}
		f.SetCellValue("adgroups_검수", "F2", model.StatusApproved)
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", res.Problems)
	}
	if len(res.Ads) != 1 {
		t.Fatalf("want 1 ad (cleared row 2 must be skipped), got %d: %+v", len(res.Ads), res.Ads)
	}
}
