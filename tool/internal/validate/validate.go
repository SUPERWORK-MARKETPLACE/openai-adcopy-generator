// Package validate implements the hard format rules from charter §6.
package validate

import "adcopy/internal/model"

type Finding struct {
	Entity  string `json:"entity"` // "campaigns" | "adgroups" | "ads"
	ID      string `json:"id"`     // campaign_name / adgroup_name / ad_name
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type Report struct {
	OK       bool      `json:"ok"`
	Errors   []Finding `json:"errors"`
	Warnings []Finding `json:"warnings"`
}

func Validate(g *model.Generated) *Report {
	r := &Report{Errors: []Finding{}, Warnings: []Finding{}}
	textFindings(g, r)
	structFindings(g, r)
	policyFindings(g, r)
	r.OK = len(r.Errors) == 0
	return r
}
