package validate

import (
	"testing"

	"adcopy/internal/model"
)

// (a) banned term이 copy에 있으면 rule "banned_term" 오류
func TestBannedTermInCopy(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"무조건"}}
	g.Ads[0].Copy = "무조건 합격 보장, 지금 신청하세요"
	if !hasRule(Validate(g).Errors, "banned_term") {
		t.Fatal("want banned_term for copy")
	}
}

// (b) banned term이 title에 있으면 오류
func TestBannedTermInTitle(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"보장"}}
	g.Ads[0].Title = "합격 보장 프로그램"
	if !hasRule(Validate(g).Errors, "banned_term") {
		t.Fatal("want banned_term for title")
	}
}

// (c) banned term이 adgroup keyword에 있으면 오류
func TestBannedTermInKeyword(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"최저가"}}
	g.Adgroups[0].Keywords[0].Text = "업계 최저가 비교"
	if !hasRule(Validate(g).Errors, "banned_term") {
		t.Fatal("want banned_term for keyword")
	}
}

// banned term 공백 트림·빈 항목 skip
func TestBannedTermBlankSkipped(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"  ", ""}}
	if hasRule(Validate(g).Errors, "banned_term") {
		t.Fatal("blank banned terms must be skipped")
	}
}

// (d) policy=nil이면 정책 오류 0 (하위호환)
func TestNoPolicyNoError(t *testing.T) {
	g := baseGenerated()
	if hasRule(Validate(g).Errors, "banned_term") {
		t.Fatal("no policy → no policy errors")
	}
}

// (e) 완전 클린 케이스 rep.OK=true
func TestPolicyCleanCaseOK(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"보장", "최저가"}}
	r := Validate(g)
	if !r.OK {
		t.Fatalf("clean policy case must be OK, got %v", rules(r.Errors))
	}
}
