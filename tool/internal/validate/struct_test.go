package validate

import (
	"strings"
	"testing"

	"adcopy/internal/model"
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

func TestAdgroupNameFormatWarning(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_학부모_문제정의" // 3 slots < 4
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	r := Validate(g)
	if !hasRule(r.Warnings, "adgroup_name_format") {
		t.Fatal("want adgroup_name_format warning for 3 slots")
	}
	if hasRule(r.Errors, "adgroup_name_format") {
		t.Fatal("adgroup_name_format must be a warning, never an error")
	}
	// 슬롯 수 부족은 퍼널 슬롯 판정 이전 단계 — 토큰 오류로 번지지 않는다.
	if hasRule(r.Errors, "adgroup_name_funnel_token") {
		t.Fatal("slot-count warning must not raise adgroup_name_funnel_token")
	}
	// 퍼널 슬롯 토큰 위반은 ERROR — adgroup_name은 업로드 파일에 그대로 실린다(9차).
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_반복훈련필요_문제발견" // 고정 토큰 아님
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	r = Validate(g)
	if !hasRule(r.Errors, "adgroup_name_funnel_token") {
		t.Fatal("want adgroup_name_funnel_token error for non-fixed funnel token")
	}
	if hasRule(r.Warnings, "adgroup_name_funnel_token") {
		t.Fatal("adgroup_name_funnel_token must be an error, never a warning")
	}
	g = baseGenerated() // fixture is already 상품/SKU_타깃_세부의도_구매여정
	r = Validate(g)
	if hasRule(r.Warnings, "adgroup_name_format") || hasRule(r.Errors, "adgroup_name_funnel_token") {
		t.Fatal("clean D10-format name must not warn")
	}
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_반복훈련필요_문제정의_2" // 중복 순번 접미
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	r = Validate(g)
	if hasRule(r.Warnings, "adgroup_name_format") || hasRule(r.Errors, "adgroup_name_funnel_token") {
		t.Fatal("numeric dedup suffix must pass — funnel token judged on previous slot")
	}
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_학부모_문제정의_2" // 실질 3슬롯 + 순번 접미
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	if !hasRule(Validate(g).Warnings, "adgroup_name_format") {
		t.Fatal("numeric suffix must not count as a content slot — want 4-slot warning")
	}
}

func TestGenerationBasisFunnelToken(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=탐색발견; 메시지=효과"
	if !hasRule(Validate(g).Warnings, "generation_basis_funnel_token") {
		t.Fatal("want generation_basis_funnel_token for non-fixed funnel name")
	}
	g = baseGenerated()
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=②제품발견; 메시지=효과"
	if hasRule(Validate(g).Warnings, "generation_basis_funnel_token") {
		t.Fatal("fixed token (with circled-number prefix) must pass")
	}
	g = baseGenerated()
	g.Adgroups[0].Trace.GenerationBasis = "SKU=훈련앱" // 퍼널 항목 없음 → 검사 제외
	if hasRule(Validate(g).Warnings, "generation_basis_funnel_token") {
		t.Fatal("trace without 퍼널= must be skipped")
	}
}

func TestGenerationBasisFunnelMissing(t *testing.T) {
	// 근거는 적었는데 퍼널 항목만 빠진 경우 → 경고 (9차 — 단계 분류가 조용히 빔)
	g := baseGenerated()
	g.Adgroups[0].Trace.GenerationBasis = "SKU=훈련앱; 메시지=효과"
	r := Validate(g)
	if !hasRule(r.Warnings, "generation_basis_funnel_missing") {
		t.Fatal("want generation_basis_funnel_missing for basis without 퍼널=")
	}
	if hasRule(r.Errors, "generation_basis_funnel_missing") {
		t.Fatal("generation_basis_funnel_missing must be a warning, never an error")
	}
	g = baseGenerated()
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 메시지=효과"
	if !hasRule(Validate(g).Warnings, "generation_basis_funnel_missing") {
		t.Fatal("want generation_basis_funnel_missing for an ad basis without 퍼널=")
	}
	g = baseGenerated()
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=제품발견; 메시지=효과"
	if hasRule(Validate(g).Warnings, "generation_basis_funnel_missing") {
		t.Fatal("basis with 퍼널= must pass")
	}
	// generation_basis 자체가 공란 → 이 규칙 대상 아님(근거 기록 여부는 별도 영역)
	if hasRule(Validate(baseGenerated()).Warnings, "generation_basis_funnel_missing") {
		t.Fatal("blank generation_basis must be exempt")
	}
}

func TestFunnelStageMerged(t *testing.T) {
	// 신청전환 그룹에 사용도움 광고가 섞이면 그룹당 1건 경고 (9차)
	secondAd := func(g *model.Generated, adName, basis string) {
		ad := g.Ads[0]
		ad.AdName = adName
		ad.Title = "완전히 다른 두 번째 광고 제목"
		ad.Copy = "퍼널 검사를 위해 내용을 완전히 바꾼 두 번째 카피예요"
		ad.Trace.GenerationBasis = basis
		g.Ads = append(g.Ads, ad)
	}
	g := baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_무료체험신청_신청전환"
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=사용도움"
	secondAd(g, "KID_01_002", "SKU=훈련앱; 퍼널=사용도움") // 같은 그룹 → 경고는 여전히 1건
	r := Validate(g)
	if n := countRule(r.Warnings, "funnel_stage_merged"); n != 1 {
		t.Fatalf("funnel_stage_merged = %d건, want 1 (그룹당 1회)", n)
	}
	if hasRule(r.Errors, "funnel_stage_merged") {
		t.Fatal("funnel_stage_merged must be a warning, never an error")
	}
	// 그룹 자신의 basis가 사용도움, 이름 슬롯이 신청전환 → 경고
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_무료체험신청_신청전환"
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	g.Adgroups[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=사용도움"
	if !hasRule(Validate(g).Warnings, "funnel_stage_merged") {
		t.Fatal("want funnel_stage_merged when the group basis and name slot split the two stages")
	}
	// 같은 단계만 있으면 미발생
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_무료체험신청_신청전환"
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=신청전환"
	if hasRule(Validate(g).Warnings, "funnel_stage_merged") {
		t.Fatal("single-stage group must not warn")
	}
	// 문제정의+제품발견 병합은 헌장 §5가 허용 → 미발생(오탐 금지)
	g = baseGenerated() // 이름 슬롯 = 제품발견
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=문제정의"
	if hasRule(Validate(g).Warnings, "funnel_stage_merged") {
		t.Fatal("문제정의+제품발견 merge is allowed — must not warn")
	}
	// 사용도움 단독 그룹도 미발생
	g = baseGenerated()
	g.Adgroups[0].AdgroupName = "훈련앱_초등학부모_학습습관관리_사용도움"
	g.Ads[0].AdgroupName = g.Adgroups[0].AdgroupName
	g.Ads[0].Trace.GenerationBasis = "SKU=훈련앱; 퍼널=사용도움"
	if hasRule(Validate(g).Warnings, "funnel_stage_merged") {
		t.Fatal("사용도움-only group must not warn")
	}
}

func countRule(fs []Finding, rule string) int {
	n := 0
	for _, f := range fs {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

func TestSourceExcerptMissing(t *testing.T) {
	// 브리프 출처 + 발췌 공란 → 경고 (F4 — 근거 추적)
	g := baseGenerated()
	g.Ads[0].Trace.SourceType = "브리프"
	if !hasRule(Validate(g).Warnings, "source_excerpt_missing") {
		t.Fatal("want source_excerpt_missing for sourced ad without excerpt")
	}
	// 브리프 출처 + 발췌 있음 → 통과
	g.Ads[0].Trace.SourceExcerpt = "6대 영역 매일 훈련, 무료 레벨테스트 제공"
	if hasRule(Validate(g).Warnings, "source_excerpt_missing") {
		t.Fatal("sourced ad with excerpt must pass")
	}
	// ai_inferred + 발췌 공란 → 통과 (맥락 추론 전용, 발췌 의무 없음)
	g = baseGenerated()
	g.Ads[0].Trace.SourceType = "ai_inferred"
	if hasRule(Validate(g).Warnings, "source_excerpt_missing") {
		t.Fatal("ai_inferred trace must be exempt")
	}
	// source_type 공란 → 통과 (검사 대상 아님)
	if hasRule(Validate(baseGenerated()).Warnings, "source_excerpt_missing") {
		t.Fatal("blank source_type must be exempt")
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

func TestAdNameFormatWarning(t *testing.T) {
	// 순번(끝 숫자) 앞 프리픽스는 광고그룹마다 고유해야 한다 — SKU 프리픽스
	// 하나로 순번 범위만 나눠 그룹을 구분하면(KID_01_001~009 → 그룹 3개) 경고.
	second := func(g *model.Generated, adName string) {
		ag := g.Adgroups[0]
		ag.AdgroupName = "훈련앱_초등학부모_학습앱비교_비교검토"
		g.Adgroups = append(g.Adgroups, ag)
		ad := g.Ads[0]
		ad.AdName = adName
		ad.AdgroupName = ag.AdgroupName
		ad.Title = "완전히 다른 두 번째 광고 제목"
		ad.Copy = "형식 검사를 위해 내용을 완전히 바꾼 두 번째 카피예요"
		g.Ads = append(g.Ads, ad)
	}
	g := baseGenerated()
	second(g, "KID_01_002") // Ads[0]=KID_01_001과 프리픽스 공유, 그룹은 다름
	r := Validate(g)
	if !hasRule(r.Warnings, "ad_name_format") {
		t.Fatal("want ad_name_format when one prefix spans two adgroups")
	}
	if hasRule(r.Errors, "ad_name_format") {
		t.Fatal("ad_name_format must be a warning, never an error")
	}
	g = baseGenerated() // 같은 그룹 안에서는 프리픽스 공유 정상
	ad := g.Ads[0]
	ad.AdName = "KID_01_002"
	ad.Title = "완전히 다른 두 번째 광고 제목"
	ad.Copy = "형식 검사를 위해 내용을 완전히 바꾼 두 번째 카피예요"
	g.Ads = append(g.Ads, ad)
	if hasRule(Validate(g).Warnings, "ad_name_format") {
		t.Fatal("ads in the same adgroup may share a prefix")
	}
	g = baseGenerated() // 그룹마다 다른 프리픽스 — 통과
	g.Ads[0].AdName = "KID_01A_001"
	second(g, "KID_01B_001")
	if hasRule(Validate(g).Warnings, "ad_name_format") {
		t.Fatal("per-adgroup prefixes must pass")
	}
	g = baseGenerated() // creative 순번(끝 숫자) 누락
	g.Ads[0].AdName = "KID_01_ABC"
	if !hasRule(Validate(g).Warnings, "ad_name_format") {
		t.Fatal("want ad_name_format for a name without a trailing creative sequence")
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
