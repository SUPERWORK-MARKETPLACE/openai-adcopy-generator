package review

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/model"
	"adcopy/internal/validate"
)

func sampleGenerated() *model.Generated {
	return &model.Generated{
		Campaigns: []model.Campaign{{CampaignName: "01_학습자료", BudgetMax: 25000,
			BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
			Objective: "Views", TargetCountries: []string{"KR"}}},
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "01_훈련앱",
			// 검색어형 2/5 = 40% — S2 권장 범위(30~50%) 안.
			Keywords: []model.Keyword{
				{Text: "초등 영어 앱 추천", Origin: "customer_data"},
				{Text: "초등 영어 단어 어플", Origin: "ai_inferred"},
				{Text: "아이 발음 교정 어떻게 시작할까", Origin: "ai_inferred"},
				{Text: "매일 영어 훈련 습관 들이려면 뭐가 좋을까", Origin: "ai_inferred"},
				{Text: "아이가 영어 단어를 자꾸 까먹을 때", Origin: "ai_inferred"},
			},
			Trace: model.Trace{SourceType: "브리프", GenerationBasis: "퍼널=제품발견", ConfidenceScore: 0.9}}},
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
	if err := WriteReview(sampleGenerated(), nil, out); err != nil {
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
	if rows[0][0] != "ad_name" || rows[0][2] != "title" ||
		rows[0][8] != "validation_status" || rows[0][9] != StatusColumnHeader || rows[0][10] != "review_comment" {
		t.Fatalf("ads_검수 headers wrong: %v", rows[0])
	}
	// validation_status carries the AI-written auto-check note; 검수상태 stays
	// blank even on a trigger row (0715 요청 1 — no pre-fill, 광고주가 기입).
	if rows[1][0] != "KID_01_001" || rows[1][8] != model.StatusNeedsAdvertiser {
		t.Fatalf("ads_검수 row wrong: %v", rows[1])
	}
	if rows[1][9] != "" {
		t.Fatalf("trigger row 검수상태 must stay blank, got %q", rows[1][9])
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
	// Clean row (validation 통과) → 검수상태 stays blank (D4: 자동 무수정 승인 없음).
	if agRows[1][5] != "" {
		t.Fatalf("clean row 검수상태 must stay blank, got %q", agRows[1][5])
	}

	dvs, err := f.GetDataValidations("ads_검수")
	if err != nil || len(dvs) == 0 {
		t.Fatalf("ads_검수 must have a status dropdown, got %v (%v)", dvs, err)
	}
	if dvs[0].Sqref != "J2:J2" {
		t.Errorf("dropdown must sit on the 검수상태 column J, got %q", dvs[0].Sqref)
	}
	// 0715 요청 1: 광고주 확인 필요 is validation_status-only — the 검수상태
	// dropdown offers the 4 advertiser statuses and nothing else. The positive
	// assertion guards against excelize returning an empty Formula1.
	if !strings.Contains(dvs[0].Formula1, model.StatusApproved) ||
		!strings.Contains(dvs[0].Formula1, model.StatusRegenerate) {
		t.Errorf("검수상태 dropdown must list the 4 statuses, got %q", dvs[0].Formula1)
	}
	if strings.Contains(dvs[0].Formula1, model.StatusNeedsAdvertiser) {
		t.Errorf("검수상태 dropdown must not contain %q: %q", model.StatusNeedsAdvertiser, dvs[0].Formula1)
	}

	// 요약 시트: 상태값 목록 행 + 상태값별 설명 행(4종 + 빈칸) — 항목 3 후속.
	sumRows, err := f.GetRows("요약")
	if err != nil {
		t.Fatal(err)
	}
	if len(sumRows) < 8+len(model.AllStatuses)+1 {
		t.Fatalf("요약 rows = %d, want >= %d (status legend rows)", len(sumRows), 8+len(model.AllStatuses)+1)
	}
	if sumRows[7][0] != "상태값" {
		t.Fatalf("요약 8행 = %q, want 상태값", sumRows[7][0])
	}
	for i, st := range model.AllStatuses {
		row := sumRows[8+i]
		if row[0] != "상태값: "+st || len(row) < 2 || row[1] == "" {
			t.Fatalf("요약 %d행 = %v, want %q + 설명", 9+i, row, "상태값: "+st)
		}
	}
	blank := sumRows[8+len(model.AllStatuses)]
	if blank[0] != "상태값: (빈칸)" || len(blank) < 2 || blank[1] == "" {
		t.Fatalf("요약 빈칸 상태 설명 행 = %v, want 상태값: (빈칸) + 설명", blank)
	}
}

// TestWriteReviewMergesValidationFindings: validation_status must show the
// per-row automatic check results (spec 6-11: 형식·정책·사실성 결과 확인).
func TestWriteReviewMergesValidationFindings(t *testing.T) {
	g := sampleGenerated()
	// Warning-only row: no AI trace note, only a machine finding.
	g.Ads = append(g.Ads, model.Ad{AdName: "KID_01_002", AdgroupName: "01_훈련앱",
		Title: "매일 10분 영어 훈련 루틴",
		Copy:  "발음과 단어를 매일 10분씩 반복하는 훈련 루틴",
		Link:  "https://www.example.com/promo", ImageLink: "https://img.example.com/a.png"})
	// Error-only row: no AI trace note, only a machine error.
	g.Ads = append(g.Ads, model.Ad{AdName: "KID_01_003", AdgroupName: "01_훈련앱",
		Title: "영어 훈련 시작 가이드",
		Copy:  "너무 길게 쓴 카피라서 최대 글자수를 넘긴다고 가정하는 예시 문장입니다",
		Link:  "https://www.example.com/promo", ImageLink: "https://img.example.com/a.png"})
	rep := &validate.Report{
		Warnings: []validate.Finding{
			{Entity: "ads", ID: "KID_01_001",
				Field: "copy", Rule: "copy_len_recommended", Message: "카피 39자 — 권장 32~36자"},
			{Entity: "ads", ID: "KID_01_002",
				Field: "copy", Rule: "copy_len_recommended", Message: "카피 24자 — 권장 32~36자"},
		},
		Errors: []validate.Finding{
			{Entity: "ads", ID: "KID_01_003",
				Field: "copy", Rule: "copy_max_48", Message: "카피 49자 — 최대 48자"},
		},
	}
	out := filepath.Join(t.TempDir(), "review.xlsx")
	if err := WriteReview(g, rep, out); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenFile(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("ads_검수")
	if err != nil {
		t.Fatal(err)
	}
	cell := rows[1][8]
	if !strings.Contains(cell, model.StatusNeedsAdvertiser) ||
		!strings.Contains(cell, "copy_len_recommended") || !strings.Contains(cell, "경고") {
		t.Fatalf("validation_status must merge trace note and finding, got %q", cell)
	}
	// 경고-단독 행 (trace note 없음): the trigger token "경고" lands in
	// validation_status, but 검수상태 stays blank (0715 요청 1 — no pre-fill).
	if !strings.Contains(rows[2][8], "경고") || rows[2][9] != "" {
		t.Fatalf("warning-only row: want 경고 in validation_status and blank 검수상태, got status=%q cell=%q",
			rows[2][9], rows[2][8])
	}
	// 오류-단독 행: the machine error shows in validation_status; 검수상태 blank.
	if !strings.Contains(rows[3][8], "오류") || rows[3][9] != "" {
		t.Fatalf("error-only row: want 오류 in validation_status and blank 검수상태, got status=%q cell=%q",
			rows[3][9], rows[3][8])
	}
	// Row with no findings and no trace note → 통과.
	agRows, err := f.GetRows("adgroups_검수")
	if err != nil {
		t.Fatal(err)
	}
	if agRows[1][4] != "통과" {
		t.Errorf("clean row validation_status = %q, want 통과", agRows[1][4])
	}
}
