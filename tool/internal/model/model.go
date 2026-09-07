// Package model defines the generated.json contract between Claude and the CLI.
package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Review status values (charter §7). StatusNeedsAdvertiser is a
// validation_status flag only — it is NOT a 검수상태 dropdown value
// (0715 요청 1: 검수상태는 광고주 검수 결과 입력란, 광고주 확인 필요 제외).
const (
	StatusApproved        = "무수정 승인"
	StatusApprovedEdited  = "수정 후 승인"
	StatusRejected        = "사용 불가"
	StatusRegenerate      = "부분 재생성"
	StatusNeedsAdvertiser = "광고주 확인 필요"
)

// AllStatuses: 검수상태 dropdown values in order (광고주 판정 4종).
var AllStatuses = []string{StatusApproved, StatusApprovedEdited, StatusRejected, StatusRegenerate}

// FunnelStages are the six fixed purchase-journey tokens (R1). adgroup_name's
// last slot (or the slot before a numeric dedup suffix) must be one of these.
var FunnelStages = []string{"문제정의", "제품발견", "비교검토", "단일제품평가", "신청전환", "사용도움"}

// FunnelFromBasis extracts the 퍼널= value recorded in generation_basis.
// ok=false when the trace has no 퍼널= entry. 값은 구분자 `;,|` 앞까지 자르고
// 원문자 접두(①~⑥)를 떼어낸다. validate의 형식 검사와 review의 분포 집계가
// 같은 규칙을 쓰도록 여기 한 곳에만 둔다.
func FunnelFromBasis(basis string) (string, bool) {
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

// AdgroupNameSlots splits adgroup_name on "_" and drops a purely numeric dedup
// suffix — 순번 접미는 내용 슬롯이 아니다(상품_타깃_구매여정_2는 3슬롯).
func AdgroupNameSlots(name string) []string {
	slots := strings.Split(name, "_")
	if n := len(slots); n > 1 && isNumericSlot(slots[n-1]) {
		slots = slots[:n-1]
	}
	return slots
}

// AdgroupFunnelSlot returns the 구매여정 슬롯 of adgroup_name — the last content
// slot per AdgroupNameSlots.
func AdgroupFunnelSlot(name string) string {
	slots := AdgroupNameSlots(name)
	return slots[len(slots)-1]
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

// ReviewTriggerTokens: if a merged validation_status cell contains any of
// these, the row is an automatic-check flag row (R6) — counted in the review
// summary and surfaced as flagged_approved when overridden with 무수정 승인.
// 8 tokens — "길이 권장 초과" and "문장 자연성 확인 필요" added by S1 (2026-07-14).
var ReviewTriggerTokens = []string{
	"의미 중복 후보", "경고", StatusNeedsAdvertiser, "길이 초과", "근거 확인 필요", "정책 확인 필요",
	"길이 권장 초과", "문장 자연성 확인 필요",
}

// HasReviewTrigger reports whether the merged validation_status cell value
// carries an automatic review flag (R6 trigger). 검수상태 is never pre-filled
// from this (0715 요청 1: 검수상태는 항상 빈칸 디폴트, 광고주가 기입) — it only
// drives the summary flag count and review-in's flagged_approved surfacing.
// Machine findings merged by review-out carry "경고(rule): msg" / "오류(rule): msg"
// prefixes; an error-only row must trigger too (it is worse than a warning row),
// so the "오류(" prefix is matched alongside the R6 trigger tokens.
func HasReviewTrigger(mergedCell string) bool {
	if strings.Contains(mergedCell, "오류(") {
		return true
	}
	for _, tok := range ReviewTriggerTokens {
		if strings.Contains(mergedCell, tok) {
			return true
		}
	}
	return false
}

type Generated struct {
	Campaigns []Campaign `json:"campaigns"`
	Adgroups  []Adgroup  `json:"adgroups"`
	Ads       []Ad       `json:"ads"`
	Policy    *Policy    `json:"policy,omitempty"`
}

// Policy holds machine-checkable ad-policy rules (charter §6/§7).
// BannedTerms merges the input policy sheet's shared banned expressions
// with per-SKU banned expressions into one global list.
type Policy struct {
	BannedTerms []string `json:"banned_terms,omitempty"`
}

type Campaign struct {
	CampaignName    string   `json:"campaign_name"`
	BudgetMax       float64  `json:"budget_max"`
	BudgetType      string   `json:"budget_type"`
	LaunchDate      string   `json:"launch_date"` // YYYY-MM-DD
	EndDate         string   `json:"end_date"`    // YYYY-MM-DD
	Objective       string   `json:"objective"`
	TargetCountries []string `json:"target_countries"`
}

type Keyword struct {
	Text   string `json:"text"`
	Origin string `json:"origin"` // "customer_data" | "ai_inferred"
}

type Adgroup struct {
	CampaignName string    `json:"campaign_name"`
	AdgroupName  string    `json:"adgroup_name"`
	MaxBid       any       `json:"max_bid,omitempty"` // must stay nil — presence is a validation error
	Keywords     []Keyword `json:"keywords"`
	Trace        Trace     `json:"trace"`
}

type Ad struct {
	AdName      string `json:"ad_name"` // internal tracking only, excluded from export
	AdgroupName string `json:"adgroup_name"`
	Title       string `json:"title"`
	Copy        string `json:"copy"`
	Link        string `json:"link"`
	ImageLink   string `json:"image_link"`
	Trace       Trace  `json:"trace"`
}

type Trace struct {
	SourceType       string  `json:"source_type"`
	SourceURL        string  `json:"source_url"`
	SourceExcerpt    string  `json:"source_excerpt"`
	GenerationBasis  string  `json:"generation_basis"`
	ConfidenceScore  float64 `json:"confidence_score"`
	ValidationStatus string  `json:"validation_status"`
	ReviewComment    string  `json:"review_comment"`
	ExclusionReason  string  `json:"exclusion_reason"`
}

func Load(path string) (*Generated, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var g Generated
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &g, nil
}
