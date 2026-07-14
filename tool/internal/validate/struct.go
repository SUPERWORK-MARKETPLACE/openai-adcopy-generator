package validate

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"adcopy/internal/model"
)

func structFindings(g *model.Generated, r *Report) {
	campaigns := map[string]bool{}
	for _, c := range g.Campaigns {
		campaigns[c.CampaignName] = true
		if strings.TrimSpace(c.CampaignName) == "" {
			r.err("campaigns", c.CampaignName, "campaign_name", "required_field", "campaign_name이 비어 있습니다")
		}
		if c.BudgetMax <= 0 {
			r.err("campaigns", c.CampaignName, "budget_max", "budget_max_positive", "budget_max는 양수여야 합니다")
		}
		for _, f := range [][2]string{{"budget_type", c.BudgetType}, {"objective", c.Objective}} {
			if strings.TrimSpace(f[1]) == "" {
				r.err("campaigns", c.CampaignName, f[0], "required_field", f[0]+"이(가) 비어 있습니다")
			}
		}
		ld, err1 := time.Parse("2006-01-02", c.LaunchDate)
		ed, err2 := time.Parse("2006-01-02", c.EndDate)
		if err1 != nil {
			r.err("campaigns", c.CampaignName, "launch_date", "date_invalid", "launch_date 형식은 YYYY-MM-DD: "+c.LaunchDate)
		}
		if err2 != nil {
			r.err("campaigns", c.CampaignName, "end_date", "date_invalid", "end_date 형식은 YYYY-MM-DD: "+c.EndDate)
		}
		if err1 == nil && err2 == nil && ed.Before(ld) {
			r.err("campaigns", c.CampaignName, "end_date", "date_order", "end_date가 launch_date보다 빠릅니다")
		}
		if len(c.TargetCountries) == 0 {
			r.err("campaigns", c.CampaignName, "target_countries", "countries_required", "target_countries가 비어 있습니다")
		}
	}

	adgroups := map[string]bool{}
	for _, ag := range g.Adgroups {
		name := ag.AdgroupName
		n := utf8.RuneCountInString(name)
		switch {
		case strings.TrimSpace(name) == "":
			r.err("adgroups", name, "adgroup_name", "adgroup_name_blank", "adgroup_name이 공백입니다")
		case n < 3 || n > 1000:
			r.err("adgroups", name, "adgroup_name", "adgroup_name_length", fmt.Sprintf("adgroup_name %d자 — 3~1000자", n))
		}
		if strings.TrimSpace(name) != "" {
			checkAdgroupNameFormat(r, name)
		}
		if adgroups[name] {
			r.err("adgroups", name, "adgroup_name", "adgroup_name_duplicate", "adgroup_name 중복")
		}
		adgroups[name] = true
		if !campaigns[ag.CampaignName] {
			r.err("adgroups", name, "campaign_name", "ref_campaign_missing", "campaigns에 없는 캠페인: "+ag.CampaignName)
		}
		if ag.MaxBid != nil {
			r.err("adgroups", name, "max_bid", "max_bid_must_be_empty",
				"max_bid는 항상 빈칸이어야 합니다(값을 넣으면 업로드 오류 — 업로드 후 시스템에서 수동 설정)")
		}
		if ag.Trace.ConfidenceScore < 0 || ag.Trace.ConfidenceScore > 1 {
			r.err("adgroups", name, "confidence_score", "confidence_range", "confidence_score는 0~1")
		}
		checkFunnelToken(r, "adgroups", name, ag.Trace.GenerationBasis)
	}

	adNames := map[string]bool{}
	for _, ad := range g.Ads {
		if strings.TrimSpace(ad.AdName) == "" {
			r.err("ads", ad.AdName, "ad_name", "ad_name_required", "ad_name이 비어 있습니다")
		} else if adNames[ad.AdName] {
			r.err("ads", ad.AdName, "ad_name", "ad_name_duplicate", "ad_name 중복")
		}
		adNames[ad.AdName] = true
		if !adgroups[ad.AdgroupName] {
			r.err("ads", ad.AdName, "adgroup_name", "ref_adgroup_missing", "adgroups에 없는 그룹: "+ad.AdgroupName)
		}
		checkURL(r, ad.AdName, "link", ad.Link)
		checkURL(r, ad.AdName, "image_link", ad.ImageLink)
		if ad.Trace.ConfidenceScore < 0 || ad.Trace.ConfidenceScore > 1 {
			r.err("ads", ad.AdName, "confidence_score", "confidence_range", "confidence_score는 0~1")
		}
		checkFunnelToken(r, "ads", ad.AdName, ad.Trace.GenerationBasis)
	}
}

// checkFunnelToken warns when generation_basis records a 퍼널= value that is
// not one of the six fixed R1 tokens. Skips traces without a 퍼널= entry.
func checkFunnelToken(r *Report, entity, id, basis string) {
	i := strings.Index(basis, "퍼널=")
	if i < 0 {
		return
	}
	v := basis[i+len("퍼널="):]
	if j := strings.IndexAny(v, ";,|"); j >= 0 {
		v = v[:j]
	}
	v = strings.TrimLeft(strings.TrimSpace(v), "①②③④⑤⑥")
	for _, s := range model.FunnelStages {
		if v == s {
			return
		}
	}
	r.warn(entity, id, "generation_basis", "generation_basis_funnel_token",
		fmt.Sprintf("퍼널=%q — 고정 토큰(%s) 중 하나여야 합니다", v, strings.Join(model.FunnelStages, "·")))
}

// checkAdgroupNameFormat enforces the R2 naming structure
// 상품/SKU_타깃_세부의도_구매여정 as a WARNING only (never blocks export).
// Split on "_": 4+ slots required; the funnel slot is the last one — or the
// one before a purely numeric dedup suffix — and must be a fixed R1 token
// (model.FunnelStages).
func checkAdgroupNameFormat(r *Report, name string) {
	slots := strings.Split(name, "_")
	// A purely numeric dedup suffix is not a content slot — strip it before
	// the minimum-slot check so 상품_타깃_구매여정_2 still warns.
	if n := len(slots); n > 1 && isNumericSlot(slots[n-1]) {
		slots = slots[:n-1]
	}
	if len(slots) < 4 {
		r.warn("adgroups", name, "adgroup_name", "adgroup_name_format",
			fmt.Sprintf("adgroup_name %d슬롯 — 형식 상품/SKU_타깃_세부의도_구매여정(4슬롯 이상)", len(slots)))
		return
	}
	funnel := slots[len(slots)-1]
	for _, s := range model.FunnelStages {
		if funnel == s {
			return
		}
	}
	r.warn("adgroups", name, "adgroup_name", "adgroup_name_format",
		fmt.Sprintf("구매여정 슬롯 %q — 고정 토큰(%s) 중 하나여야 합니다", funnel, strings.Join(model.FunnelStages, "·")))
}

func isNumericSlot(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func checkURL(r *Report, id, field, raw string) {
	if strings.TrimSpace(raw) == "" {
		r.err("ads", id, field, "required_field", field+"이(가) 비어 있습니다")
		return
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		r.err("ads", id, field, "url_invalid", field+" 형식 오류: "+raw)
		return
	}
	if u.Scheme != "https" {
		r.warn("ads", id, field, "url_not_https", field+"가 https가 아닙니다: "+raw)
	}
}
