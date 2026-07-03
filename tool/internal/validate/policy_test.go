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

// (d) required_phrase가 title+copy에 있으면 통과
func TestRequiredPhrasePresent(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].RequiredPhrases = []string{"무료학습"}
	// baseGenerated copy already contains "무료학습"
	if hasRule(Validate(g).Errors, "required_phrase_missing") {
		t.Fatal("phrase present in copy must pass")
	}
}

// required_phrase가 title에 있어도 통과 (combined)
func TestRequiredPhraseInTitle(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Title = "공식 인증 프로그램"
	g.Adgroups[0].RequiredPhrases = []string{"공식 인증"}
	if hasRule(Validate(g).Errors, "required_phrase_missing") {
		t.Fatal("phrase present in title must pass")
	}
}

// (e) required_phrase가 없으면 "required_phrase_missing" 오류
func TestRequiredPhraseMissing(t *testing.T) {
	g := baseGenerated()
	g.Adgroups[0].RequiredPhrases = []string{"존재하지않는문구"}
	if !hasRule(Validate(g).Errors, "required_phrase_missing") {
		t.Fatal("want required_phrase_missing")
	}
}

// (f) policy=nil이고 required_phrases 없으면 정책 오류 0 (하위호환)
func TestNoPolicyNoError(t *testing.T) {
	g := baseGenerated()
	for _, f := range Validate(g).Errors {
		if f.Rule == "banned_term" || f.Rule == "required_phrase_missing" {
			t.Fatalf("no policy → no policy errors, got %s", f.Rule)
		}
	}
}

// (g) 완전 클린 케이스 rep.OK=true
func TestPolicyCleanCaseOK(t *testing.T) {
	g := baseGenerated()
	g.Policy = &model.Policy{BannedTerms: []string{"보장", "최저가"}}
	g.Adgroups[0].RequiredPhrases = []string{"무료학습"}
	r := Validate(g)
	if !r.OK {
		t.Fatalf("clean policy case must be OK, got %v", rules(r.Errors))
	}
}
