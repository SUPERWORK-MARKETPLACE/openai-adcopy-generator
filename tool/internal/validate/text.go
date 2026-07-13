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
			if strings.HasSuffix(body, "세요") {
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

	seenHint := map[string]string{} // campaign+"\x00"+hint -> first adgroup_name
	for _, ag := range g.Adgroups {
		if len(ag.Keywords) < keywordsMin {
			r.err("adgroups", ag.AdgroupName, "keywords", "keywords_min_5",
				fmt.Sprintf("Context Hints %d개 — 최소 %d개", len(ag.Keywords), keywordsMin))
		}
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
			// R4: exact-duplicate hints across different adgroups of one campaign.
			key := ag.CampaignName + "\x00" + text
			if first, dup := seenHint[key]; dup {
				if first != ag.AdgroupName {
					r.warn("adgroups", ag.AdgroupName, "keywords", "keyword_cross_adgroup_duplicate",
						fmt.Sprintf("힌트 %q — 같은 캠페인의 %s와(과) 중복", text, first))
				}
			} else {
				seenHint[key] = ag.AdgroupName
			}
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

func (r *Report) err(entity, id, field, rule, msg string) {
	r.Errors = append(r.Errors, Finding{entity, id, field, rule, msg})
}

func (r *Report) warn(entity, id, field, rule, msg string) {
	r.Warnings = append(r.Warnings, Finding{entity, id, field, rule, msg})
}
