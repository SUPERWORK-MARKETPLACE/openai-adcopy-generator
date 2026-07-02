package validate

import (
	"strings"
	"testing"
)

func TestAdgroupNameRules(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].AdgroupName = "aa" // 2 runes < 3
	g.Ads[0].AdgroupName = "aa"
	if !hasRule(Validate(g).Errors, "adgroup_name_length") {
		t.Fatal("want adgroup_name_length for 2 runes")
	}
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "   "
	g.Ads[0].AdgroupName = "   "
	if !hasRule(Validate(g).Errors, "adgroup_name_blank") {
		t.Fatal("want adgroup_name_blank")
	}
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = strings.Repeat("가", 1001)
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	if !hasRule(Validate(g).Errors, "adgroup_name_length") {
		t.Fatal("want adgroup_name_length for 1001 runes")
	}
	g = baseGenerated()
	g.Adgroups = append(g.Adgroups, g.Adgroups[0]) // duplicate name
	if !hasRule(Validate(g).Errors, "adgroup_name_duplicate") {
		t.Fatal("want adgroup_name_duplicate")
	}
}

func TestAdNameRules(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].AdName = ""
	if !hasRule(Validate(g).Errors, "ad_name_required") {
		t.Fatal("want ad_name_required")
	}
	g = baseGenerated()
	dup := g.Ads[0]
	dup.Title = "다른 제목으로 바꾼 중복 이름 광고" // avoid creative_duplicate noise
	dup.Copy = "본문도 완전히 다른 내용으로 채운 두 번째 광고 카피입니다"
	g.Ads = append(g.Ads, dup)
	if !hasRule(Validate(g).Errors, "ad_name_duplicate") {
		t.Fatal("want ad_name_duplicate")
	}
}

func TestReferentialIntegrity(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].CampaignName = "없는캠페인"
	if !hasRule(Validate(g).Errors, "ref_campaign_missing") {
		t.Fatal("want ref_campaign_missing")
	}
	g = baseGenerated()
	g.Ads[0].AdgroupName = "없는그룹"
	if !hasRule(Validate(g).Errors, "ref_adgroup_missing") {
		t.Fatal("want ref_adgroup_missing")
	}
}

func TestMaxBidMustBeEmpty(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].MaxBid = 1000.0
	if !hasRule(Validate(g).Errors, "max_bid_must_be_empty") {
		t.Fatal("want max_bid_must_be_empty — 값이 있으면 업로드 오류")
	}
}

func TestCampaignFieldRules(t *testing.T) {
	g := baseGenerated()
	g.Campaigns[0].BudgetMax = 0
	if !hasRule(Validate(g).Errors, "budget_max_positive") {
		t.Fatal("want budget_max_positive")
	}
	g = baseGenerated()
	g.Campaigns[0].LaunchDate = "2026-13-01"
	if !hasRule(Validate(g).Errors, "date_invalid") {
		t.Fatal("want date_invalid")
	}
	g = baseGenerated()
	g.Campaigns[0].LaunchDate, g.Campaigns[0].EndDate = "2026-08-01", "2026-07-01"
	if !hasRule(Validate(g).Errors, "date_order") {
		t.Fatal("want date_order")
	}
	g = baseGenerated()
	g.Campaigns[0].TargetCountries = nil
	if !hasRule(Validate(g).Errors, "countries_required") {
		t.Fatal("want countries_required")
	}
	g = baseGenerated()
	g.Campaigns[0].Objective = ""
	if !hasRule(Validate(g).Errors, "required_field") {
		t.Fatal("want required_field for empty objective")
	}
}

func TestURLFormat(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Link = "notaurl"
	if !hasRule(Validate(g).Errors, "url_invalid") {
		t.Fatal("want url_invalid for link")
	}
	g = baseGenerated()
	g.Ads[0].ImageLink = ""
	if !hasRule(Validate(g).Errors, "required_field") {
		t.Fatal("want required_field for empty image_link")
	}
}

func TestConfidenceRange(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Trace.ConfidenceScore = 1.5
	if !hasRule(Validate(g).Errors, "confidence_range") {
		t.Fatal("want confidence_range")
	}
}
