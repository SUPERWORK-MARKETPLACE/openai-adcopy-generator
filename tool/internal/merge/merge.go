// Package merge combines per-batch generation chunk files into one Generated
// document for the bulk-generation mode.
package merge

import (
	"fmt"
	"reflect"

	"adcopy/internal/model"
)

// Merge loads chunk files and combines them: campaigns are deduplicated by
// campaign_name (definitions must be identical across chunks), adgroups/ads
// are concatenated in input order, and policy.banned_terms are unioned
// (order-preserving dedup) so the banned-word check survives bulk merge.
// Cross-chunk name duplicates are left for validate to catch.
func Merge(paths []string) (*model.Generated, error) {
	out := &model.Generated{}
	seen := map[string]model.Campaign{}
	bannedSeen := map[string]bool{}
	var banned []string
	for _, p := range paths {
		g, err := model.Load(p)
		if err != nil {
			return nil, err
		}
		for _, c := range g.Campaigns {
			prev, ok := seen[c.CampaignName]
			if !ok {
				seen[c.CampaignName] = c
				out.Campaigns = append(out.Campaigns, c)
				continue
			}
			if !reflect.DeepEqual(prev, c) {
				return nil, fmt.Errorf("%s: 캠페인 %q 정의가 다른 청크와 충돌합니다", p, c.CampaignName)
			}
		}
		out.Adgroups = append(out.Adgroups, g.Adgroups...)
		out.Ads = append(out.Ads, g.Ads...)
		if g.Policy != nil {
			for _, t := range g.Policy.BannedTerms {
				if !bannedSeen[t] {
					bannedSeen[t] = true
					banned = append(banned, t)
				}
			}
		}
	}
	if len(banned) > 0 {
		out.Policy = &model.Policy{BannedTerms: banned}
	}
	return out, nil
}
