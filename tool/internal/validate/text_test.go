package validate

import (
	"fmt"
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
		Adgroups: []model.Adgroup{{CampaignName: "01_학습자료", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			// 검색어형 2/5 = 40% — S2 권장 범위(30~50%) 안.
			Keywords: kw("초등 영어 어떻게 시작할까", "아이 영어 흥미 붙이려면 뭐가 좋을까",
				"초등 영어 반복 훈련 앱 추천", "초등 영어 단어 암기 앱",
				"아이가 영어를 자꾸 까먹을 때")}},
		Ads: []model.Ad{{AdName: "KID_01_001", AdgroupName: "훈련앱_초등학부모_반복훈련필요_제품발견",
			Title: "초등 영어 반복 훈련이 필요하다면", // 17 runes: recommended band
			Copy:  "6대 영역을 매일 훈련하고 무료 레벨테스트로 진단할 수 있어요", // 34 runes, complete sentence, non-CTA ending
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

func TestKeywordCrossAdgroupDuplicate(t *testing.T) {
	g := baseGenerated()
	// 공백만 다른 변형("초등영어" vs "초등 영어")도 중복으로 잡아야 한다.
	g.Adgroups = append(g.Adgroups, model.Adgroup{
		CampaignName: "01_학습자료", AdgroupName: "훈련앱_초등학부모_학습시작고민_문제정의",
		Keywords: kw("초등영어 어떻게 시작할까", "다른 힌트는 뭐가 좋을까", "셋째 힌트 어떨까",
			"넷째 힌트 어떨까요", "다섯째 힌트 어떨까요")})
	if !hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("want keyword_cross_adgroup_duplicate for whitespace-variant dup across groups")
	}
}

func TestKeywordDuplicateGlobalScope(t *testing.T) {
	// 같은 광고그룹 내부 반복은 이 규칙 대상이 아니다
	g := baseGenerated()
	g.Adgroups[0].Keywords = kw("초등 영어 어떻게 시작할까", "초등 영어 어떻게 시작할까",
		"셋째 힌트 어떨까", "넷째 힌트 어떨까요", "다섯째 힌트 어떨까요")
	if hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("in-group repetition must not trigger the cross-group rule")
	}
	// 범용 문구는 캠페인이 달라도 잡는다(전역 스코프 — R4 범용 문구 반복 금지)
	g = baseGenerated()
	g.Campaigns = append(g.Campaigns, model.Campaign{CampaignName: "02_다른캠페인", BudgetMax: 1000,
		BudgetType: "daily", LaunchDate: "2026-07-01", EndDate: "2026-07-31",
		Objective: "Views", TargetCountries: []string{"KR"}})
	g.Adgroups = append(g.Adgroups, model.Adgroup{
		CampaignName: "02_다른캠페인", AdgroupName: "훈련앱_초등학부모_학습시작고민_문제정의",
		Keywords: kw("초등 영어 어떻게 시작할까", "다른 힌트는 뭐가 좋을까", "셋째 힌트 어떨까",
			"넷째 힌트 어떨까요", "다섯째 힌트 어떨까요")})
	if !hasRule(Validate(g).Warnings, "keyword_cross_adgroup_duplicate") {
		t.Fatal("generic phrase repeated across campaigns must warn (global scope)")
	}
}

func TestKeywordSearchformRatio(t *testing.T) {
	// S2: 검색어형 40%±10(30~50%) 밖이면 양측 모두 경고.
	// mkHints: 검색어형 search개 + 문장형 sentence개.
	mkHints := func(search, sentence int) []model.Keyword {
		var texts []string
		for i := 0; i < search; i++ {
			texts = append(texts, fmt.Sprintf("초등 영어 교재 추천 유형%d", i+1))
		}
		for i := 0; i < sentence; i++ {
			texts = append(texts, fmt.Sprintf("아이 영어 %d단계는 어떻게 시작할까", i+1))
		}
		return kw(texts...)
	}
	// 검색어형 1/5 = 20% < 30% → 경고 (하한)
	g := baseGenerated()
	g.Adgroups[0].Keywords = kw("초등 영어 학습지 추천", "영어 공부가 막막할 때",
		"집에서 영어 시작해도 될까", "영어 흥미 붙이려면 뭐가 좋을까", "아이가 영어를 자꾸 까먹을 때")
	if !hasRule(Validate(g).Warnings, "keyword_searchform_ratio") {
		t.Fatal("want keyword_searchform_ratio when search-form hints are below 30%")
	}
	// 포함 경계: 정확히 30%(3/10)·50%(5/10)는 통과, 20%(2/10)·60%(6/10)는 경고
	for _, tc := range []struct {
		search, sentence int
		warn             bool
	}{{2, 8, true}, {3, 7, false}, {5, 5, false}, {6, 4, true}} {
		g = baseGenerated()
		g.Adgroups[0].Keywords = mkHints(tc.search, tc.sentence)
		got := hasRule(Validate(g).Warnings, "keyword_searchform_ratio")
		if got != tc.warn {
			t.Fatalf("search-form %d/%d: warn=%v, want %v", tc.search, tc.search+tc.sentence, got, tc.warn)
		}
	}
	// 검색어형 2/5 = 40% → 경고 없음
	g = baseGenerated()
	g.Adgroups[0].Keywords = kw("초등 영어 학습지 추천", "영어 단어 어플 추천",
		"아이 영어 어떻게 시작할까", "영어 공부가 막막할 때", "집에서 영어 시작해도 될까")
	if hasRule(Validate(g).Warnings, "keyword_searchform_ratio") {
		t.Fatal("40% search-form must not warn")
	}
	// 검색어형 3/5 = 60% > 50% → 경고 (상한)
	g = baseGenerated()
	g.Adgroups[0].Keywords = kw("초등 영어 학습지 추천", "영어 단어 어플 추천", "초등 영어 무료 교재",
		"아이 영어 어떻게 시작할까", "영어 공부가 막막할 때")
	if !hasRule(Validate(g).Warnings, "keyword_searchform_ratio") {
		t.Fatal("want keyword_searchform_ratio when search-form hints exceed 50%")
	}
	// 영어 힌트는 판정 제외 — 영어만 있으면 경고 없음
	g = baseGenerated()
	g.Adgroups[0].Keywords = kw("english learning app", "kids english practice",
		"phonics for beginners", "daily english routine", "english reading habit")
	if hasRule(Validate(g).Warnings, "keyword_searchform_ratio") {
		t.Fatal("English-only hints must be exempt from the ratio check")
	}
}

func TestCopySentenceForm(t *testing.T) {
	msg := func(fs []Finding) string {
		for _, f := range fs {
			if f.Rule == "copy_sentence_form" {
				return f.Message
			}
		}
		return ""
	}
	// 명사구 종결 → 경고
	g := baseGenerated()
	g.Ads[0].Copy = "무료 레벨테스트와 매일 10분씩 하는 영어 반복 훈련 루틴"
	if m := msg(Validate(g).Warnings); !strings.Contains(m, "명사구") {
		t.Fatalf("want 명사구 warning for noun-phrase ending, got %q", m)
	}
	// 조건절 종결(…다면) → 경고
	g = baseGenerated()
	g.Ads[0].Copy = "매일 10분 훈련으로 아이 영어 습관을 만들어 주고 싶다면"
	if m := msg(Validate(g).Warnings); !strings.Contains(m, "조건절") {
		t.Fatalf("want 조건절 warning for conditional-clause ending, got %q", m)
	}
	// 완결 문장(…있어요) → 통과 (fixture copy)
	if hasRule(Validate(baseGenerated()).Warnings, "copy_sentence_form") {
		t.Fatal("complete sentence (…있어요) must pass")
	}
	// 완결 질문(…할까요?) → 통과
	g = baseGenerated()
	g.Ads[0].Copy = "우리 아이 영어, 매일 10분 반복 훈련으로 시작해 볼까요?"
	if hasRule(Validate(g).Warnings, "copy_sentence_form") {
		t.Fatal("complete question (…?) must pass")
	}
	// 마침표 붙은 완결 문장(…세요.) → 통과 (꼬리 구두점 제거 경로)
	g = baseGenerated()
	g.Ads[0].Copy = "무료 레벨테스트로 아이 영어 수준을 먼저 확인해보세요."
	if hasRule(Validate(g).Warnings, "copy_sentence_form") {
		t.Fatal("sentence ending in …세요. must pass")
	}
	// 감탄 종결(…!) → 통과 (조기 통과 경로)
	g = baseGenerated()
	g.Ads[0].Copy = "신청 후 7일간 주요 콘텐츠를 무료로 경험할 수 있어요!"
	if hasRule(Validate(g).Warnings, "copy_sentence_form") {
		t.Fatal("copy ending in ! must pass")
	}
	// 영어 카피는 판정 제외
	g = baseGenerated()
	g.Ads[0].Copy = "Start daily English training with a free level test"
	if hasRule(Validate(g).Warnings, "copy_sentence_form") {
		t.Fatal("English copy must be exempt")
	}
}

func TestCopySpeechLevel(t *testing.T) {
	// 반말 평서형(…있다) → 경고 (F3)
	g := baseGenerated()
	g.Ads[0].Copy = "학부모 앱으로 아이의 학습 데이터를 직접 확인해볼 수 있다"
	if !hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("want copy_speech_level for 반말 …있다 ending")
	}
	// 반말 평서형 + 꼬리 구두점(…된다.) → 경고
	g = baseGenerated()
	g.Ads[0].Copy = "체험 후 마음에 들면 정회원, 아니면 그대로 반납하면 된다."
	if !hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("want copy_speech_level for punctuated 반말 …된다. ending")
	}
	// 합니다체(…합니다) → 통과
	g = baseGenerated()
	g.Ads[0].Copy = "6대 영역을 매일 훈련하고 무료 레벨테스트로 진단합니다"
	if hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("합니다체 ending must pass")
	}
	// 해요체(…있어요) → 통과 (fixture copy)
	if hasRule(Validate(baseGenerated()).Warnings, "copy_speech_level") {
		t.Fatal("해요체 ending must pass")
	}
	// …답니다 → 통과 (니다 계열)
	g = baseGenerated()
	g.Ads[0].Copy = "아이들이 매일 10분씩 즐겁게 영어 훈련을 이어간답니다"
	if hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("…답니다 ending must pass")
	}
	// 어간이 '니다' 룬으로 끝나는 반말(…아니다) → 경고 (합니다체 오인 금지)
	g = baseGenerated()
	g.Ads[0].Copy = "학습기 반납은 어렵지 않고 추가 비용 부담도 전혀 아니다"
	if !hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("want copy_speech_level for 반말 …아니다 ending")
	}
	// 영어 카피는 판정 제외
	g = baseGenerated()
	g.Ads[0].Copy = "Start daily English training with a free level test"
	if hasRule(Validate(g).Warnings, "copy_speech_level") {
		t.Fatal("English copy must be exempt")
	}
}

func TestTitleCopyOverlap(t *testing.T) {
	// 제목을 그대로 풀어 쓴 카피 → 경고 (F2 실사례)
	g := baseGenerated()
	g.Ads[0].Title = "학부모 앱으로 보는 학습 데이터"
	g.Ads[0].Copy = "학부모 앱으로 아이의 학습 데이터를 직접 확인할 수 있어요"
	if !hasRule(Validate(g).Warnings, "title_copy_overlap") {
		t.Fatal("want title_copy_overlap for near-verbatim repetition")
	}
	// 정상 쌍(제목에 없는 새 정보) → 통과
	g = baseGenerated()
	g.Ads[0].Title = "아이 영어 수준부터 알고 싶다면"
	g.Ads[0].Copy = "집에서 미리 체험해 보면 아이 수준과 흥미를 알 수 있어요"
	if hasRule(Validate(g).Warnings, "title_copy_overlap") {
		t.Fatal("normal pair must pass")
	}
	// 짧은 제목(bigram 6개 미만)은 판정 제외 — 카피가 제목을 그대로 포함해도 통과
	g = baseGenerated()
	g.Ads[0].Title = "무료 체험 신청" // 6자 → bigram 5개 < 6
	g.Ads[0].Copy = "무료 체험 신청 후 아이 영어 수준을 바로 확인해 보세요"
	if hasRule(Validate(g).Warnings, "title_copy_overlap") {
		t.Fatal("short title must be exempt")
	}
}

func TestCopyCtaRatioOver30Warns(t *testing.T) {
	g := baseGenerated()
	g.Ads[0].Copy = "아이에게 맞는 영어 학습법을 지금 무료로 확인해보세요" // CTA ending, 1/1 = 100%
	if !hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("want copy_cta_ratio_30 for 100% CTA endings")
	}
}

func TestCopyCtaRatioBoundary(t *testing.T) {
	g := baseGenerated()
	mk := func(n int, copyText string) model.Ad {
		ad := g.Ads[0]
		ad.AdName = fmt.Sprintf("KID_01_%03d", n)
		ad.Title = fmt.Sprintf("서로 다른 제목 %d", n)
		ad.Copy = copyText
		return ad
	}
	// 1 CTA / 4 copies = 25% → 경고 없음
	g.Ads = []model.Ad{
		mk(1, "무료 레벨테스트로 먼저 진단"),
		mk(2, "집에서 시작하는 초등 영어 루틴"),
		mk(3, "학원과 온라인 학습, 차이를 비교"),
		mk(4, "아이 수준에 맞는 학습법을 확인해보세요"),
	}
	if hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("25% CTA must not warn")
	}
	// 2 CTA / 4 copies = 50% → 경고
	g.Ads[1] = mk(2, "지금 무료체험을 신청하세요")
	if !hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("50% CTA must warn")
	}
	// 3 CTA / 10 copies = 정확히 30% → 경고 없음(초과만 경고)
	g = baseGenerated()
	base := g.Ads[0]
	g.Ads = nil
	for i := 0; i < 10; i++ {
		ad := base
		ad.AdName = fmt.Sprintf("CTA_%02d", i)
		ad.Title = fmt.Sprintf("경계 검증용 제목 %02d", i)
		ad.Copy = "집에서 시작하는 초등 영어 루틴"
		if i < 3 {
			ad.Copy = "지금 무료체험을 신청하세요"
		}
		g.Ads = append(g.Ads, ad)
	}
	if hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("exactly 30% CTA must not warn")
	}
}

func TestCopyCtaEndingPunctuation(t *testing.T) {
	// 구두점으로 끝나는 CTA(…세요.)도 CTA로 집계해야 한다.
	g := baseGenerated()
	g.Ads[0].Copy = "아이에게 맞는 영어 학습법을 지금 확인해보세요." // 1/1 = 100%
	if !hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("want copy_cta_ratio_30 for punctuated CTA ending (…세요.)")
	}
	// 질문형(…세요?)은 CTA가 아니다.
	g = baseGenerated()
	g.Ads[0].Copy = "아이 영어 학습 시작이 아직도 고민되지 않으세요?"
	if hasRule(Validate(g).Warnings, "copy_cta_ratio_30") {
		t.Fatal("question form (…세요?) must not count as CTA")
	}
}
