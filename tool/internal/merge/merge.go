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
// are concatenated in input order. Cross-chunk name duplicates are left for
// validate to catch.
func Merge(paths []string) (*model.Generated, error) {
	out := &model.Generated{}
	seen := map[string]model.Campaign{}
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
	}
	return out, nil
}
