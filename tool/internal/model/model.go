// Package model defines the generated.json contract between Claude and the CLI.
package model

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Review status values (charter §7).
const (
	StatusApproved        = "무수정 승인"
	StatusApprovedEdited  = "수정 후 승인"
	StatusRejected        = "사용 불가"
	StatusRegenerate      = "부분 재생성"
	StatusNeedsAdvertiser = "광고주 확인 필요"
)

// AllStatuses in dropdown order.
var AllStatuses = []string{StatusApproved, StatusApprovedEdited, StatusRejected, StatusRegenerate, StatusNeedsAdvertiser}

// FunnelStages are the six fixed purchase-journey tokens (R1). adgroup_name's
// last slot (or the slot before a numeric dedup suffix) must be one of these.
var FunnelStages = []string{"문제정의", "제품발견", "비교검토", "단일제품평가", "신청전환", "사용도움"}

// ReviewTriggerTokens: if a merged validation_status cell contains any of
// these, the operator-decision default must be StatusNeedsAdvertiser (R6).
// 8 tokens — "길이 권장 초과" and "문장 자연성 확인 필요" added by S1 (2026-07-14).
var ReviewTriggerTokens = []string{
	"의미 중복 후보", "경고", StatusNeedsAdvertiser, "길이 초과", "근거 확인 필요", "정책 확인 필요",
	"길이 권장 초과", "문장 자연성 확인 필요",
}

// NeedsAdvertiserDefault reports whether the merged validation_status cell
// value triggers the default operator decision "광고주 확인 필요".
// Machine findings merged by review-out carry "경고(rule): msg" / "오류(rule): msg"
// prefixes; an error-only row must trigger too (it is worse than a warning row),
// so the "오류(" prefix is matched alongside the R6 trigger tokens.
func NeedsAdvertiserDefault(mergedCell string) bool {
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
