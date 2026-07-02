package review

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
)

func sampleGenerated() *model.Generated {
	return &model.Generated{
		Campaigns: []model.Campaign{{CampaignName: "01_학습자료", BudgetMax: 25000,
			BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
			Objective: "Views", TargetCountries: []string{"KR"}}},
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "01_훈련앱",
			Keywords: []model.Keyword{
				{Text: "초등 영어 앱 추천", Origin: "customer_data"},
				{Text: "영어 단어 어플", Origin: "ai_inferred"},
				{Text: "발음 교정 프로그램", Origin: "ai_inferred"},
				{Text: "영어 훈련 앱", Origin: "ai_inferred"},
				{Text: "단어 퀴즈 앱", Origin: "ai_inferred"},
			},
			Trace: model.Trace{SourceType: "브리프", GenerationBasis: "퍼널=제품 발견", ConfidenceScore: 0.9}}},
		Ads: []model.Ad{{AdName: "KID_01_001", AdgroupName: "01_훈련앱",
			Title: "초등 영어 반복 훈련이 필요하다면",
			Copy:  "6대 영역 재미있고 다양하게 매일 훈련, 무료학습 신청해 보세요",
			Link:  "https://www.example.com/promo", ImageLink: "https://img.example.com/a.png",
			Trace: model.Trace{SourceType: "브리프", ConfidenceScore: 0.8,
				ValidationStatus: model.StatusNeedsAdvertiser}}},
	}
}

func TestWriteReviewWorkbook(t *testing.T) {
	out := filepath.Join(t.TempDir(), "review.xlsx")
	if err := WriteReview(sampleGenerated(), out); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	want := []string{"요약", "adgroups_검수", "ads_검수"}
	got := f.GetSheetList()
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("sheets = %v, want %v", got, want)
	}

	rows, err := f.GetRows("ads_검수")
	if err != nil {
		t.Fatal(err)
	}
	if rows[0][0] != "ad_name" || rows[0][2] != "title" || rows[0][8] != "validation_status" {
		t.Fatalf("ads_검수 headers wrong: %v", rows[0])
	}
	if rows[1][0] != "KID_01_001" || rows[1][8] != model.StatusNeedsAdvertiser {
		t.Fatalf("ads_검수 row wrong: %v", rows[1])
	}
	if rows[1][3] != "18" { // title rune count as string
		t.Errorf("title_글자수 = %q, want 18", rows[1][3])
	}

	agRows, err := f.GetRows("adgroups_검수")
	if err != nil {
		t.Fatal(err)
	}
	var kws []string
	if err := json.Unmarshal([]byte(agRows[1][2]), &kws); err != nil || len(kws) != 5 {
		t.Fatalf("keywords cell must be a JSON array of 5: %q (%v)", agRows[1][2], err)
	}
	var origins []string
	if err := json.Unmarshal([]byte(agRows[1][3]), &origins); err != nil || origins[0] != "customer_data" {
		t.Fatalf("keywords_origin cell wrong: %q", agRows[1][3])
	}

	dvs, err := f.GetDataValidations("ads_검수")
	if err != nil || len(dvs) == 0 {
		t.Fatalf("ads_검수 must have a status dropdown, got %v (%v)", dvs, err)
	}
}
