package validate

import (
	"fmt"
	"strings"
	"testing"

	"adcopy/internal/model"
)

func kw(texts ...string) []model.Keyword {
	var out []model.Keyword
	for _, t := range texts {
		out = append(out, model.Keyword{Text: t, Origin: "ai_inferred"})
	}
	return out
}

func baseGenerated() *model.Generated {
	return &model.Generated{
		Campaigns: []model.Campaign{{CampaignName: "01_학습자료", BudgetMax: 25000,
			BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
			Objective: "Views", TargetCountries: []string{"KR"}}},
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			Keywords: kw("힌트 하나", "힌트 둘", "힌트 셋", "힌트 넷", "힌트 다섯")}},
		Ads: []model.Ad{{AdName: "KID_01_001", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			Title: "초등 영어 반복 훈련이 필요하다면", // 17 runes: recommended band
			Copy:  "6대 영역 재미있고 다양하게 매일 훈련, 무료 레벨테스트 진단", // 34 runes, non-CTA ending
			Link:  "https://www.example.com/promo", ImageLink: "https://img.example.com/a.png"}},
	}
}

func rules(fs []Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Rule)
	}
	return out
}

func hasRule(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func TestCleanInputHasNoTextErrors(t *testing.T) {
	r := Validate(baseGenerated())
	if !r.OK || len(r.Errors) != 0 {
		t.Fatalf("want OK, got errors: %v", rules(r.Errors))
	}
}

func TestTitleTooLong(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Title = strings.Repeat("가", 25) // 25 runes > 24
	r := Validate(g)
	if !hasRule(r.Errors, "title_max_24") {
		t.Fatalf("want title_max_24, got %v", rules(r.Errors))
	}
	if r.OK {
		t.Fatal("report must not be OK")
	}
}

func TestTitleAt24IsBoundaryOK(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Title = strings.Repeat("가", 24)
	r := Validate(g)
	if hasRule(r.Errors, "title_max_24") {
		t.Fatal("24 runes must pass the max rule")
	}
	if !hasRule(r.Warnings, "title_len_recommended") {
		t.Fatalf("24 runes is outside 16~18: want warning, got %v", rules(r.Warnings))
	}
}

func TestCopyTooLongAndEmpty(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Copy = strings.Repeat("나", 49)
	if !hasRule(Validate(g).Errors, "copy_max_48") {
		t.Fatal("want copy_max_48")
	}
	g.Ads[0].Copy = "   "
	if !hasRule(Validate(g).Errors, "copy_required") {
		t.Fatal("want copy_required for blank copy")
	}
}

func TestCopyEqualsTitle(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Copy = g.Ads[0].Title
	if !hasRule(Validate(g).Errors, "copy_equals_title") {
		t.Fatal("want copy_equals_title")
	}
}

func TestDuplicateCreative(t *testing.T) {
	g := baseGenerated()
	dup := g.Ads[0]
	dup.AdName = "KID_01_002"
	g.Ads = append(g.Ads, dup)
	if !hasRule(Validate(g).Errors, "creative_duplicate") {
		t.Fatal("want creative_duplicate for same title+copy pair")
	}
}

func TestKeywordsMinFive(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].Keywords = kw("하나", "둘", "셋", "넷")
	if !hasRule(Validate(g).Errors, "keywords_min_5") {
		t.Fatal("want keywords_min_5")
	}
}

func TestKeywordOriginValidated(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].Keywords[0].Origin = "guess"
	if !hasRule(Validate(g).Errors, "keyword_origin_invalid") {
		t.Fatal("want keyword_origin_invalid")
	}
	g.Adgroups[0].Keywords[0] = model.Keyword{Text: " ", Origin: "ai_inferred"}
	if !hasRule(Validate(g).Errors, "keyword_empty") {
		t.Fatal("want keyword_empty")
	}
}

func TestKeywordCrossAdgroupDuplicate(t *testing.T) {
	g := baseGenerated()
	g.Adgroups = append(g.Adgroups, model.Adgroup{
		CampaignName: "01_학습자료", AdgroupName: "훈련앱_초등학부모_학습시작고민_문제정의",
		Keywords: kw("힌트 하나 ", "다른 하나", "다른 둘", "다른 셋", "다른 넷")}) // "힌트 하나" dup after trim
	if !hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("want keyword_cross_adgroup_duplicate for same hint across groups in one campaign")
	}
}

func TestKeywordDuplicateScopedToCrossGroupSameCampaign(t *testing.T) {
	// 같은 광고그룹 내부 반복은 이 규칙 대상이 아니다
	g := baseGenerated()
	g.Adgroups[0].Keywords = kw("힌트 하나", "힌트 하나", "힌트 셋", "힌트 넷", "힌트 다섯")
	if hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("in-group repetition must not trigger the cross-group rule")
	}
	// 다른 캠페인의 같은 힌트도 대상이 아니다
	g = baseGenerated()
	g.Campaigns = append(g.Campaigns, model.Campaign{CampaignName: "02_다른캠페인", BudgetMax: 1000,
		BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
		Objective: "Views", TargetCountries: []string{"KR"}})
	g.Adgroups = append(g.Adgroups, model.Adgroup{
		CampaignName: "02_다른캠페인", AdgroupName: "훈련앱_초등학부모_학습시작고민_문제정의",
		Keywords: kw("힌트 하나", "다른 하나", "다른 둘", "다른 셋", "다른 넷")})
	if hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("same hint across different campaigns must not warn")
	}
}

func TestCopyCtaRatioOver30Warns(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Copy = "아이에게 맞는 영어 학습법을 지금 무료로 확인해보세요" // CTA ending, 1/1 = 100%
	if !hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("want copy_cta_ratio_30 for 100% CTA endings")
	}
}

func TestCopyCtaRatioBoundary(t *testing.T) {
	g := baseGenerated()
	mk := func(n int, copyText string) model.Ad {
		ad := g.Ads[0]
		ad.AdName = fmt.Sprintf("KID_01_%03d", n)
		ad.Title = fmt.Sprintf("서로 다른 제목 %d", n)
		ad.Copy = copyText
		return ad
	}
	// 1 CTA / 4 copies = 25% → 경고 없음
	g.Ads = []model.Ad{
		mk(1, "무료 레벨테스트로 먼저 진단"),
		mk(2, "집에서 시작하는 초등 영어 루틴"),
		mk(3, "학원과 온라인 학습, 차이를 비교"),
		mk(4, "아이 수준에 맞는 학습법을 확인해보세요"),
	}
	if hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("25% CTA must not warn")
	}
	// 2 CTA / 4 copies = 50% → 경고
	g.Ads[1] = mk(2, "지금 무료체험을 신청하세요")
	if !hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("50% CTA must warn")
	}
	// 3 CTA / 10 copies = 정확히 30% → 경고 없음(초과만 경고)
	g = baseGenerated()
	base := g.Ads[0]
	g.Ads = nil
	for i := 0; i < 10; i++ {
		ad := base
		ad.AdName = fmt.Sprintf("CTA_%02d", i)
		ad.Title = fmt.Sprintf("경계 검증용 제목 %02d", i)
		ad.Copy = "집에서 시작하는 초등 영어 루틴"
		if i < 3 {
			ad.Copy = "지금 무료체험을 신청하세요"
		}
		g.Ads = append(g.Ads, ad)
	}
	if hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("exactly 30% CTA must not warn")
	}
}
