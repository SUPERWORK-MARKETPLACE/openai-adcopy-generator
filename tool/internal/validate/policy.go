package validate

import (
	"strings"

	"adcopy/internal/model"
)

// policyFindings enforces machine-checkable ad-policy rules (charter §6/§7):
// global banned terms. ERRORs block export. Missing policy → the check is
// skipped (backward compatible).
//
// Workbook v2 replaced the verbatim "필수 포함 문구" column with "카피 핵심
// 반영 요소" (candidate elements, 1-2 picked and reworded per ad, equivalent
// meaning allowed) — that is generation-time guidance and cannot be checked
// by string matching, so no phrase check exists here.
func policyFindings(g *model.Generated, r *Report) {
	if g.Policy == nil {
		return
	}
	for _, raw := range g.Policy.BannedTerms {
		term := strings.TrimSpace(raw)
		if term == "" {
			continue
		}
		for _, ad := range g.Ads {
			if strings.Contains(ad.Title, term) {
				r.err("ads", ad.AdName, "title", "banned_term", "금지어 포함: "+term)
			}
			if strings.Contains(ad.Copy, term) {
				r.err("ads", ad.AdName, "copy", "banned_term", "금지어 포함: "+term)
			}
		}
		for _, ag := range g.Adgroups {
			for _, k := range ag.Keywords {
				if strings.Contains(k.Text, term) {
					r.err("adgroups", ag.AdgroupName, "keywords", "banned_term", "금지어 포함: "+term)
				}
			}
		}
	}
}
