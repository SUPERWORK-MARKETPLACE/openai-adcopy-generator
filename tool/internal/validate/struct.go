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
	funnelByAdgroup := map[string]map[string]bool{} // adgroup_name -> 퍼널 값 집합
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
		checkFunnelBasisMissing(r, "adgroups", name, ag.Trace.GenerationBasis)
		// 병합 감지용 퍼널 값 수집 — 이름의 구매여정 슬롯 + 그룹 자신의 basis.
		stages := map[string]bool{}
		if slots := adgroupNameSlots(name); len(slots) > 0 {
			stages[slots[len(slots)-1]] = true
		}
		if v, ok := funnelFromBasis(ag.Trace.GenerationBasis); ok {
			stages[v] = true
		}
		funnelByAdgroup[name] = stages
	}

	adNames := map[string]bool{}
	adPrefixGroup := map[string]string{} // ad_name prefix(순번 제외) -> first adgroup_name
	prefixWarned := map[string]bool{}
	for _, ad := range g.Ads {
		if strings.TrimSpace(ad.AdName) == "" {
			r.err("ads", ad.AdName, "ad_name", "ad_name_required", "ad_name이 비어 있습니다")
		} else if adNames[ad.AdName] {
			r.err("ads", ad.AdName, "ad_name", "ad_name_duplicate", "ad_name 중복")
		} else {
			checkAdNameFormat(r, ad.AdName, ad.AdgroupName, adPrefixGroup, prefixWarned)
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
		// F4: a sourced ad must carry its excerpt — 근거 추적 (§7). ai_inferred
		// traces are exempt (맥락 추론 전용, no document to quote).
		if st := strings.TrimSpace(ad.Trace.SourceType); st != "" && st != "ai_inferred" &&
			strings.TrimSpace(ad.Trace.SourceExcerpt) == "" {
			r.warn("ads", ad.AdName, "source_excerpt", "source_excerpt_missing",
				"출처 발췌(source_excerpt)가 비어 있습니다 — 근거 추적 필수")
		}
		checkFunnelToken(r, "ads", ad.AdName, ad.Trace.GenerationBasis)
		checkFunnelBasisMissing(r, "ads", ad.AdName, ad.Trace.GenerationBasis)
		if v, ok := funnelFromBasis(ad.Trace.GenerationBasis); ok {
			if stages := funnelByAdgroup[ad.AdgroupName]; stages != nil {
				stages[v] = true
			}
		}
	}

	checkFunnelStageMerged(r, g, funnelByAdgroup)
}

// 판단 목적이 서로 다른 두 단계 — 한 광고그룹에 섞이면 경고한다(9차 요구사항).
const (
	funnelConversion = "신청전환" // 핵심 행동 실행 직전 판단
	funnelUsageHelp  = "사용도움" // 이용·경험 정보 탐색
)

// funnelFromBasis extracts the 퍼널= value recorded in generation_basis.
// ok=false when the trace has no 퍼널= entry. 값은 구분자 `;,|` 앞까지 자르고
// 원문자 접두(①~⑥)를 떼어낸다.
func funnelFromBasis(basis string) (string, bool) {
	i := strings.Index(basis, "퍼널=")
	if i < 0 {
		return "", false
	}
	v := basis[i+len("퍼널="):]
	if j := strings.IndexAny(v, ";,|"); j >= 0 {
		v = v[:j]
	}
	return strings.TrimLeft(strings.TrimSpace(v), "①②③④⑤⑥"), true
}

// checkFunnelToken warns when generation_basis records a 퍼널= value that is
// not one of the six fixed R1 tokens. Skips traces without a 퍼널= entry.
func checkFunnelToken(r *Report, entity, id, basis string) {
	v, ok := funnelFromBasis(basis)
	if !ok {
		return
	}
	for _, s := range model.FunnelStages {
		if v == s {
			return
		}
	}
	r.warn(entity, id, "generation_basis", "generation_basis_funnel_token",
		fmt.Sprintf("퍼널=%q — 고정 토큰(%s) 중 하나여야 합니다", v, strings.Join(model.FunnelStages, "·")))
}

// checkFunnelBasisMissing warns when a filled generation_basis records no
// 퍼널= entry — 퍼널 값이 없으면 단계 분류·분포 점검이 조용히 비어 버린다(9차).
// generation_basis 자체가 공란인 행은 근거 기록 여부 문제라 이 규칙 대상이
// 아니다(중복 경고 방지).
func checkFunnelBasisMissing(r *Report, entity, id, basis string) {
	if strings.TrimSpace(basis) == "" {
		return
	}
	if _, ok := funnelFromBasis(basis); ok {
		return
	}
	r.warn(entity, id, "generation_basis", "generation_basis_funnel_missing",
		fmt.Sprintf("generation_basis에 퍼널= 항목이 없습니다 — 고정 토큰(%s) 중 하나를 기록하세요",
			strings.Join(model.FunnelStages, "·")))
}

// checkFunnelStageMerged warns when 신청전환 and 사용도움 end up in one
// adgroup (9차 요구사항). 두 단계는 판단 목적이 다르다 — 신청전환은 핵심 행동
// 실행 직전 판단, 사용도움은 이용·경험 정보 탐색이다. 그 밖의 단계 조합 병합은
// 헌장 §5가 허용하므로 경고하지 않는다(오탐 금지). 그룹당 1회만 보고한다.
func checkFunnelStageMerged(r *Report, g *model.Generated, funnelByAdgroup map[string]map[string]bool) {
	warned := map[string]bool{}
	for _, ag := range g.Adgroups {
		name := ag.AdgroupName
		stages := funnelByAdgroup[name]
		if warned[name] || !stages[funnelConversion] || !stages[funnelUsageHelp] {
			continue
		}
		warned[name] = true
		r.warn("adgroups", name, "adgroup_name", "funnel_stage_merged",
			fmt.Sprintf("%s과 %s이 한 광고그룹에 섞였습니다 — 두 단계는 판단 목적이 달라 분리해야 합니다"+
				"(%s=핵심 행동 실행 직전 판단, %s=이용·경험 정보 탐색)",
				funnelConversion, funnelUsageHelp, funnelConversion, funnelUsageHelp))
	}
}

// checkAdgroupNameFormat enforces the R2 naming structure
// 상품/SKU_타깃_세부의도_구매여정. Split on "_": 4+ slots required; the funnel
// slot is the last one — or the one before a purely numeric dedup suffix — and
// must be a fixed R1 token (model.FunnelStages).
// 슬롯 수 부족은 WARNING(export를 막지 않는다), 퍼널 슬롯 토큰 위반은 ERROR다 —
// adgroup_name은 업로드 파일에 그대로 실리므로 임의 명칭이 최종 산출물에 남으면
// 안 된다(9차). 내부 근거 추적 필드인 generation_basis는 경고를 유지한다.
func checkAdgroupNameFormat(r *Report, name string) {
	slots := adgroupNameSlots(name)
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
	r.err("adgroups", name, "adgroup_name", "adgroup_name_funnel_token",
		fmt.Sprintf("구매여정 슬롯 %q — 고정 토큰(%s) 중 하나여야 합니다", funnel, strings.Join(model.FunnelStages, "·")))
}

// checkAdNameFormat enforces the ad_name structure 캠페인/SKU 코드 + 광고그룹
// 코드 + creative 순번 as a WARNING only. The creative sequence is the trailing
// digit run; the remaining prefix must identify a single adgroup — a prefix
// shared by two adgroups means the adgroup-code slot is missing (순번 범위로만
// 그룹을 구분하는 이름, 예: KID_01A_001~009 하나로 그룹 3개를 커버).
// Warned once per (prefix, extra adgroup) pair to avoid flagging every ad.
func checkAdNameFormat(r *Report, name, adgroupName string, firstGroup map[string]string, warned map[string]bool) {
	prefix := strings.TrimRight(name, "0123456789")
	if prefix == name {
		r.warn("ads", name, "ad_name", "ad_name_format",
			"creative 순번(끝 숫자) 없음 — 형식 캠페인/SKU 코드+광고그룹 코드+순번")
		return
	}
	first, seen := firstGroup[prefix]
	if !seen {
		firstGroup[prefix] = adgroupName
		return
	}
	if first == adgroupName || warned[prefix+"\x00"+adgroupName] {
		return
	}
	warned[prefix+"\x00"+adgroupName] = true
	r.warn("ads", name, "ad_name", "ad_name_format",
		fmt.Sprintf("프리픽스 %q가 광고그룹 %q와(과) 공유됩니다 — 광고그룹 코드로 구분 필요", prefix, first))
}

// adgroupNameSlots splits adgroup_name on "_" and drops a purely numeric dedup
// suffix — 순번 접미는 내용 슬롯이 아니다(상품_타깃_구매여정_2는 3슬롯). 구매여정
// 슬롯은 남은 마지막 슬롯이다.
func adgroupNameSlots(name string) []string {
	slots := strings.Split(name, "_")
	if n := len(slots); n > 1 && isNumericSlot(slots[n-1]) {
		slots = slots[:n-1]
	}
	return slots
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
