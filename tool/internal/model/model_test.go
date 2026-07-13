package model

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleJSON = `{
  "campaigns": [{
    "campaign_name": "01_학습자료", "budget_max": 25000, "budget_type": "daily",
    "launch_date": "2026-07-01", "end_date": "2026-07-31",
    "objective": "Views", "target_countries": ["KR"]
  }],
  "adgroups": [{
    "campaign_name": "01_학습자료", "adgroup_name": "훈련앱_초등학부모_반복훈련필요_제품발견",
    "keywords": [
      {"text": "초등 영어 말하기 연습 앱 추천", "origin": "customer_data"},
      {"text": "영어 단어 어플 추천", "origin": "ai_inferred"}
    ],
    "trace": {"source_type": "브리프", "source_url": "https://example.com",
      "source_excerpt": "발췌", "generation_basis": "SKU=훈련앱; 퍼널=제품발견",
      "confidence_score": 0.9, "validation_status": "", "review_comment": "", "exclusion_reason": ""}
  }],
  "ads": [{
    "ad_name": "KID_01_001", "adgroup_name": "훈련앱_초등학부모_반복훈련필요_제품발견",
    "title": "초등 영어 반복 훈련이 더 필요하다면?",
    "copy": "6대 영역 재미있고 다양하게 매일 훈련, 캐츠잉글리시 무료학습 확인",
    "link": "https://www.example.com/promo", "image_link": "https://img.example.com/a.png",
    "trace": {"source_type": "브리프", "source_url": "", "source_excerpt": "",
      "generation_basis": "", "confidence_score": 0.8,
      "validation_status": "", "review_comment": "", "exclusion_reason": ""}
  }]
}`

func writeSample(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "generated.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadParsesAllEntities(t *testing.T) {
	g, err := Load(writeSample(t, sampleJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Campaigns) != 1 || len(g.Adgroups) != 1 || len(g.Ads) != 1 {
		t.Fatalf("counts = %d/%d/%d, want 1/1/1", len(g.Campaigns), len(g.Adgroups), len(g.Ads))
	}
	if g.Adgroups[0].Keywords[0].Text != "초등 영어 말하기 연습 앱 추천" {
		t.Errorf("keyword text mismatch: %q", g.Adgroups[0].Keywords[0].Text)
	}
	if g.Adgroups[0].MaxBid != nil {
		t.Errorf("max_bid must be nil when absent, got %v", g.Adgroups[0].MaxBid)
	}
	if g.Campaigns[0].TargetCountries[0] != "KR" {
		t.Errorf("target_countries mismatch")
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	if _, err := Load(writeSample(t, "{broken")); err == nil {
		t.Fatal("want error for broken JSON")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestNeedsAdvertiserDefault(t *testing.T) {
	for _, tok := range ReviewTriggerTokens {
		if !NeedsAdvertiserDefault(tok) {
			t.Errorf("want true for trigger token %q", tok)
		}
	}
	if !NeedsAdvertiserDefault("경고(copy_len_recommended): 카피 30자 — 권장 32~36자") {
		t.Error("want true for merged cell containing a 경고 finding")
	}
	if !NeedsAdvertiserDefault("오류(copy_max_48): 카피 49자 — 최대 48자") {
		t.Error("want true for merged cell containing an 오류-only finding")
	}
	for _, cell := range []string{"", "통과"} {
		if NeedsAdvertiserDefault(cell) {
			t.Errorf("want false for %q", cell)
		}
	}
}
