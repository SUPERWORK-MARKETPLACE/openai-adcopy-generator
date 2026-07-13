package export

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
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			Keywords: []model.Keyword{
				{Text: "초등 영어 앱 추천", Origin: "customer_data"},
				{Text: "영어 단어 어플", Origin: "ai_inferred"},
				{Text: "발음 교정 프로그램", Origin: "ai_inferred"},
				{Text: "영어 훈련 앱", Origin: "ai_inferred"},
				{Text: "단어 퀴즈 앱", Origin: "ai_inferred"},
			}}},
		Ads: []model.Ad{{AdName: "KID_01_001", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			Title: "초등 영어 반복 훈련이 필요하다면",
			Copy:  "6대 영역 재미있고 다양하게 매일 훈련하는 무료학습 프로그램",
			Link:  "https://www.example.com/promo", ImageLink: "https://img.example.com/a.png"}},
	}
}

// goldenFixture is a de-identified copy of the real decrypted upload sheet:
// same sheet names + header rows, all cell content redacted to "SAMPLE". The
// real advertiser workbook is kept out of the repo (see .gitignore: data/).
const goldenFixture = "testdata/golden_structure.xlsx"

// Golden: exported headers must match the official upload schema exactly.
func TestExportMatchesRealSheetStructure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "final.xlsx")
	if err := Export(sampleGenerated(), out); err != nil {
		t.Fatal(err)
	}
	got, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Close()
	golden, err := excelize.OpenFile(goldenFixture)
	if err != nil {
		t.Fatalf("golden fixture missing: %v", err)
	}
	defer golden.Close()

	gs, ws := got.GetSheetList(), golden.GetSheetList()
	if len(gs) != len(ws) {
		t.Fatalf("sheet count %d, want %d", len(gs), len(ws))
	}
	for i := range ws {
		if gs[i] != ws[i] {
			t.Fatalf("sheet[%d] = %q, want %q", i, gs[i], ws[i])
		}
		gr, _ := got.GetRows(gs[i])
		wr, _ := golden.GetRows(ws[i])
		if len(gr) == 0 || len(wr) == 0 {
			t.Fatalf("sheet %s empty", ws[i])
		}
		if len(gr[0]) != len(wr[0]) {
			t.Fatalf("%s headers %v, want %v", ws[i], gr[0], wr[0])
		}
		for c := range wr[0] {
			if gr[0][c] != wr[0][c] {
				t.Fatalf("%s header[%d] = %q, want %q", ws[i], c, gr[0][c], wr[0][c])
			}
		}
	}
}

func TestExportContent(t *testing.T) {
	out := filepath.Join(t.TempDir(), "final.xlsx")
	if err := Export(sampleGenerated(), out); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ag, _ := f.GetRows("adgroups")
	// headers: campaign_name, adgroup_name, max_bid, keywords
	maxBid, err := f.GetCellValue("adgroups", "C2")
	if err != nil || maxBid != "" {
		t.Fatalf("max_bid must be empty, got %q", maxBid)
	}
	var kws []string
	if err := json.Unmarshal([]byte(ag[1][3]), &kws); err != nil || len(kws) != 5 || kws[0] != "초등 영어 앱 추천" {
		t.Fatalf("keywords cell wrong: %q", ag[1][3])
	}

	ads, _ := f.GetRows("ads")
	if len(ads[0]) != 5 || ads[0][0] != "adgroup_name" {
		t.Fatalf("ads headers must exclude ad_name: %v", ads[0])
	}
	if ads[1][1] != "초등 영어 반복 훈련이 필요하다면" {
		t.Fatalf("ads title wrong: %v", ads[1])
	}

	cd, _ := f.GetCellValue("campaigns", "D2") // launch_date formatted
	if cd == "" {
		t.Fatal("launch_date cell empty")
	}
	tc, _ := f.GetCellValue("campaigns", "G2")
	if tc != `["KR"]` {
		t.Fatalf("target_countries = %q, want [\"KR\"]", tc)
	}
}
