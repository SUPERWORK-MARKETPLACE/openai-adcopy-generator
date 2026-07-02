package validate

import (
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
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "01_훈련앱",
			Keywords: kw("힌트 하나", "힌트 둘", "힌트 셋", "힌트 넷", "힌트 다섯")}},
		Ads: []model.Ad{{AdName: "KID_01_001", AdgroupName: "01_훈련앱",
			Title: "초등 영어 반복 훈련이 필요하다면", // 17 runes: recommended band
			Copy:  "6대 영역 재미있고 다양하게 매일 훈련, 무료학습 신청해 보세요", // 33 runes
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
