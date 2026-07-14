package validate

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"adcopy/internal/model"
)

const (
	titleMax, titleRecLo, titleRecHi = 24, 16, 18
	copyMax, copyRecLo, copyRecHi    = 48, 32, 36
	keywordsMin                      = 5
)

func textFindings(g *model.Generated, r *Report) {
	seenCreative := map[string]string{} // title+"\x00"+copy -> first ad_name
	totalCopies := map[string]int{}     // adgroup_name -> non-empty copy count
	ctaCopies := map[string]int{}       // adgroup_name -> CTA-ending (…세요) copy count
	for _, ad := range g.Ads {
		title := strings.TrimSpace(ad.Title)
		body := strings.TrimSpace(ad.Copy)
		tn, cn := utf8.RuneCountInString(title), utf8.RuneCountInString(body)

		if title == "" {
			r.err("ads", ad.AdName, "title", "title_required", "제목이 비어 있습니다")
		} else if tn > titleMax {
			r.err("ads", ad.AdName, "title", "title_max_24", fmt.Sprintf("제목 %d자 — 최대 %d자", tn, titleMax))
		} else if tn < titleRecLo || tn > titleRecHi {
			r.warn("ads", ad.AdName, "title", "title_len_recommended", fmt.Sprintf("제목 %d자 — 권장 %d~%d자", tn, titleRecLo, titleRecHi))
		}

		if body != "" {
			totalCopies[ad.AdgroupName]++
			if isCTAEnding(body) {
				ctaCopies[ad.AdgroupName]++
			}
		}

		if body == "" {
			r.err("ads", ad.AdName, "copy", "copy_required", "카피가 비어 있습니다")
		} else if cn > copyMax {
			r.err("ads", ad.AdName, "copy", "copy_max_48", fmt.Sprintf("카피 %d자 — 최대 %d자", cn, copyMax))
		} else if cn < copyRecLo || cn > copyRecHi {
			r.warn("ads", ad.AdName, "copy", "copy_len_recommended", fmt.Sprintf("카피 %d자 — 권장 %d~%d자", cn, copyRecLo, copyRecHi))
		}

		// S4: a Korean copy must be a complete natural sentence — noun-phrase
		// and conditional-clause endings are flagged. Warning only.
		if body != "" && hasHangul(body) {
			if msg := sentenceFormIssue(body); msg != "" {
				r.warn("ads", ad.AdName, "copy", "copy_sentence_form", msg)
			}
		}

		if title != "" && title == body {
			r.err("ads", ad.AdName, "copy", "copy_equals_title", "카피가 제목과 동일합니다")
		}
		key := title + "\x00" + body
		if first, dup := seenCreative[key]; dup && title != "" {
			r.err("ads", ad.AdName, "title", "creative_duplicate", "제목·카피가 "+first+"와(과) 동일합니다")
		} else {
			seenCreative[key] = ad.AdName
		}
	}

	seenHint := map[string]string{} // whitespace-normalized hint -> first adgroup_name
	for _, ag := range g.Adgroups {
		if len(ag.Keywords) < keywordsMin {
			r.err("adgroups", ag.AdgroupName, "keywords", "keywords_min_5",
				fmt.Sprintf("Context Hints %d개 — 최소 %d개", len(ag.Keywords), keywordsMin))
		}
		koTotal, koSearch := 0, 0
		for i, k := range ag.Keywords {
			text := strings.TrimSpace(k.Text)
			if text == "" {
				r.err("adgroups", ag.AdgroupName, "keywords", "keyword_empty", fmt.Sprintf("%d번째 힌트가 비어 있습니다", i+1))
			}
			if k.Origin != "customer_data" && k.Origin != "ai_inferred" {
				r.err("adgroups", ag.AdgroupName, "keywords", "keyword_origin_invalid",
					fmt.Sprintf("%d번째 힌트 origin=%q — customer_data 또는 ai_inferred", i+1, k.Origin))
			}
			if text == "" {
				continue
			}
			// R4: exact-duplicate hints across adgroups. Global scope (not just
			// same campaign) — a generic phrase repeated anywhere in the workbook
			// blurs group intent. Whitespace-insensitive so spacing variants match.
			key := strings.Join(strings.Fields(text), "")
			if first, dup := seenHint[key]; dup {
				if first != ag.AdgroupName {
					r.warn("adgroups", ag.AdgroupName, "keywords", "keyword_cross_adgroup_duplicate",
						fmt.Sprintf("힌트 %q — %s와(과) 중복(범용 문구 반복 금지)", text, first))
				}
			} else {
				seenHint[key] = ag.AdgroupName
			}
			// R3·S2: track search-query-style hints among Korean hints.
			if hasHangul(text) {
				koTotal++
				if !isSentenceFormHint(text) {
					koSearch++
				}
			}
		}
		// S2: search-form hints should be 40%±10 (30~50%) of Korean hints per
		// adgroup — question/situation forms carry the rest. Both bounds warn.
		// Heuristic on Korean hints only (English hints are exempt), warning only.
		if koTotal > 0 && (koSearch*10 < koTotal*3 || koSearch*2 > koTotal) {
			r.warn("adgroups", ag.AdgroupName, "keywords", "keyword_searchform_ratio",
				fmt.Sprintf("검색어형 힌트 %d/%d — 권장 40%%±10(30~50%%)", koSearch, koTotal))
		}
		// R5: CTA-style endings (…세요) capped at 30% of copies per adgroup.
		if total := totalCopies[ag.AdgroupName]; total > 0 {
			if cta := ctaCopies[ag.AdgroupName]; cta*10 > total*3 {
				r.warn("adgroups", ag.AdgroupName, "copy", "copy_cta_ratio_30",
					fmt.Sprintf("CTA형 종결 카피 %d/%d — 광고그룹당 30%% 이하", cta, total))
			}
		}
	}
}

// isCTAEnding reports whether a copy ends in an imperative …세요 form,
// optionally followed by closing punctuation (예: "확인해보세요.").
// Question forms ("…않으세요?") are not CTA.
func isCTAEnding(body string) bool {
	if strings.HasSuffix(body, "?") || strings.HasSuffix(body, "？") {
		return false
	}
	return strings.HasSuffix(strings.TrimRight(body, ".!。！ "), "세요")
}

// sentenceFormIssue reports why a copy fails the complete-sentence rule (S4),
// or "" if it passes. A trailing ?/! is a complete question/exclamation; after
// stripping closing punctuation, a 면 ending is a dangling conditional clause,
// 요/다/까/죠 endings read as complete sentences, anything else is a
// noun-phrase/incomplete ending.
func sentenceFormIssue(body string) string {
	for _, suf := range []string{"?", "？", "!", "！"} {
		if strings.HasSuffix(body, suf) {
			return ""
		}
	}
	t := strings.TrimRight(body, ".!。！ ")
	if strings.HasSuffix(t, "면") {
		return "조건절 종결 — copy는 완결 문장이어야 합니다"
	}
	for _, suf := range []string{"요", "다", "까", "죠"} {
		if strings.HasSuffix(t, suf) {
			return ""
		}
	}
	return "명사구·불완전 종결 — copy는 완결 문장이어야 합니다"
}

// isSentenceFormHint reports whether a hint reads as a question/situation
// sentence rather than a search-query noun phrase (R3·S2). Heuristic: a
// question mark, an interrogative word, or a sentence-final ending counts as
// sentence form. Conservative on purpose — feeds a warning-only per-group ratio.
func isSentenceFormHint(text string) bool {
	t := strings.TrimSpace(text)
	if strings.HasSuffix(t, "?") || strings.HasSuffix(t, "？") {
		return true
	}
	for _, m := range []string{"어떻게", "어디서", "무엇", "뭐가", "얼마나", "언제"} {
		if strings.Contains(t, m) {
			return true
		}
	}
	t = strings.TrimRight(t, ".!。！ ")
	for _, suf := range []string{"요", "까", "죠", "다면", "라면", "은데", "인데", "때"} {
		if strings.HasSuffix(t, suf) {
			return true
		}
	}
	return false
}

func hasHangul(s string) bool {
	for _, r := range s {
		if (r >= 0xAC00 && r <= 0xD7A3) || (r >= 0x3131 && r <= 0x318E) {
			return true
		}
	}
	return false
}

func (r *Report) err(entity, id, field, rule, msg string) {
	r.Errors = append(r.Errors, Finding{entity, id, field, rule, msg})
}

func (r *Report) warn(entity, id, field, rule, msg string) {
	r.Warnings = append(r.Warnings, Finding{entity, id, field, rule, msg})
}
