package review

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

// writeEditedReview simulates the operator: fills statuses, edits one copy.
func writeEditedReview(t *testing.T, g *model.Generated, edit func(f *excelize.File)) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "review.xlsx")
	if err := WriteReview(g, p); err != nil {
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
		f.SetCellValue("ads_검수", "I2", model.StatusApprovedEdited) // validation_status
		f.SetCellValue("ads_검수", "E2", "수정된 카피 문구로 교체합니다")   // copy
		f.SetCellValue("ads_검수", "J2", "혜택 표현 수정")             // review_comment
		f.SetCellValue("adgroups_검수", "E2", model.StatusApproved)
	})
	res, err := ReadReview(p, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", res.Problems)
	}
	ad := res.Ads[0]
	if ad.ValidationStatus != model.StatusApprovedEdited || ad.Copy != "수정된 카피 문구로 교체합니다" {
		t.Fatalf("edited ad not read back: %+v", ad)
	}
	if res.Adgroups[0].ValidationStatus != model.StatusApproved {
		t.Fatalf("adgroup status not read back: %+v", res.Adgroups[0])
	}
	if len(res.Adgroups[0].Keywords) != 5 {
		t.Fatalf("keywords not parsed: %+v", res.Adgroups[0].Keywords)
	}
}

func TestReadReviewFlagsProblems(t *testing.T) {
	g := sampleGenerated()
	p := writeEditedReview(t, g, func(f *excelize.File) {
		f.SetCellValue("ads_검수", "A2", "UNKNOWN_AD")     // ad_name not in generated
		f.SetCellValue("ads_검수", "I2", "승인함")           // invalid status value
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
