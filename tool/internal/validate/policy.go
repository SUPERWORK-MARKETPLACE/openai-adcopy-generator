package validate

import (
	"strings"

	"adcopy/internal/model"
)

// policyFindings enforces machine-checkable ad-policy rules (charter §6/§7):
// global banned terms and per-adgroup required phrases. Both are ERRORs that
// block export. Missing policy / required_phrases → the check is skipped
// (backward compatible).
func policyFindings(g *model.Generated, r *Report) {
	if g.Policy != nil {
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

	for _, ag := range g.Adgroups {
		if len(ag.RequiredPhrases) == 0 {
			continue
		}
		for _, ad := range g.Ads {
			if ad.AdgroupName != ag.AdgroupName {
				continue
			}
			combined := ad.Title + " " + ad.Copy
			for _, phrase := range ag.RequiredPhrases {
				if !strings.Contains(combined, phrase) {
					r.err("ads", ad.AdName, "copy", "required_phrase_missing", "필수 문구 누락: "+phrase)
				}
			}
		}
	}
}
