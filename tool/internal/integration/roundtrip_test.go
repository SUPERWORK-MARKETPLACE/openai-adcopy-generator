package integration

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/export"
	"adcopy/internal/model"
	"adcopy/internal/review"
	"adcopy/internal/validate"
)

func fullGenerated() *model.Generated {
	kw := func(texts ...string) []model.Keyword {
		var out []model.Keyword
		for _, t := range texts {
			out = append(out, model.Keyword{Text: t, Origin: "ai_inferred"})
		}
		return out
	}
	return &model.Generated{
		Campaigns: []model.Campaign{{CampaignName: "01_학습자료", BudgetMax: 25000,
			BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
			Objective: "Views", TargetCountries: []string{"KR"}}},
		Adgroups: []model.Adgroup{
			{CampaignName: "01_학습자료", AdgroupName: "01_훈련앱",
				Keywords: kw("초등 영어 앱 추천", "영어 단어 어플", "발음 교정 프로그램", "영어 훈련 앱", "단어 퀴즈 앱")},
			{CampaignName: "01_학습자료", AdgroupName: "01_영어도서관",
				Keywords: kw("영어 동화책 추천", "온라인 영어 도서관", "영어 원서 읽기", "영어 오디오북", "리더스북 추천")},
		},
		Ads: []model.Ad{
			{AdName: "KID_01_001", AdgroupName: "01_훈련앱",
				Title: "초등 영어 반복 훈련이 필요하다면",
				Copy:  "6대 영역 재미있고 다양하게 매일 훈련, 무료학습 신청해 보세요",
				Link:  "https://www.example.com/a", ImageLink: "https://img.example.com/a.png"},
			{AdName: "KID_01_002", AdgroupName: "01_영어도서관",
				Title: "초등 원서 맞춤 큐레이팅 하는 법",
				Copy:  "이천 권 영어도서관과 매주 한 권 원서 통독을 무료로 신청해요",
				Link:  "https://www.example.com/b", ImageLink: "https://img.example.com/b.png"},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	g := fullGenerated()

	// 1. generation output must validate clean
	if rep := validate.Validate(g); !rep.OK {
		t.Fatalf("precondition: %+v", rep.Errors)
	}

	// 2. write review workbook
	reviewPath := filepath.Join(dir, "review.xlsx")
	if err := review.WriteReview(g, reviewPath); err != nil {
		t.Fatal(err)
	}

	// 3. operator: approve ad 1, reject ad 2, approve both adgroups
	f, err := excelize.OpenFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	f.SetCellValue("ads_검수", "I2", model.StatusApproved)
	f.SetCellValue("ads_검수", "I3", model.StatusRejected)
	f.SetCellValue("adgroups_검수", "E2", model.StatusApproved)
	f.SetCellValue("adgroups_검수", "E3", model.StatusApproved)
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 4. read back
	res, err := review.ReadReview(reviewPath, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("problems: %v", res.Problems)
	}

	// 5. filter approved (finalize logic Claude will apply, mirrored here)
	statusByAd := map[string]string{}
	for _, a := range res.Ads {
		statusByAd[a.AdName] = a.ValidationStatus
	}
	approved := &model.Generated{Campaigns: g.Campaigns, Adgroups: g.Adgroups}
	usedAdgroups := map[string]bool{}
	for _, ad := range g.Ads {
		s := statusByAd[ad.AdName]
		if s == model.StatusApproved || s == model.StatusApprovedEdited {
			approved.Ads = append(approved.Ads, ad)
			usedAdgroups[ad.AdgroupName] = true
		}
	}
	var keptAdgroups []model.Adgroup
	for _, ag := range approved.Adgroups {
		if usedAdgroups[ag.AdgroupName] {
			keptAdgroups = append(keptAdgroups, ag)
		}
	}
	approved.Adgroups = keptAdgroups

	// 6. approved subset must validate and export
	if rep := validate.Validate(approved); !rep.OK {
		t.Fatalf("approved subset invalid: %+v", rep.Errors)
	}
	finalPath := filepath.Join(dir, "final.xlsx")
	if err := export.Export(approved, finalPath); err != nil {
		t.Fatal(err)
	}

	out, err := excelize.OpenFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	ads, _ := out.GetRows("ads")
	if len(ads) != 2 { // header + approved ad only
		t.Fatalf("ads rows = %d, want 2 (rejected ad excluded)", len(ads))
	}
	if ads[1][1] != "초등 영어 반복 훈련이 필요하다면" {
		t.Fatalf("surviving ad must be the approved KID_01_001's creative, got row %v", ads[1])
	}
	ags, _ := out.GetRows("adgroups")
	if len(ags) != 2 { // header + used adgroup only
		t.Fatalf("adgroups rows = %d, want 2 (unused adgroup dropped)", len(ags))
	}
	if ags[1][1] != "01_훈련앱" {
		t.Fatalf("surviving adgroup must be 01_훈련앱, got row %v", ags[1])
	}
}
