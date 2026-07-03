package merge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"adcopy/internal/model"
)

func writeChunk(t *testing.T, dir, name string, g *model.Generated) string {
	t.Helper()
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func camp(name string) model.Campaign {
	return model.Campaign{CampaignName: name, BudgetMax: 25000, BudgetType: "daily",
		LaunchDate: "2026-07-01", EndDate: "2026-07-31", Objective: "Views",
		TargetCountries: []string{"KR"}}
}

func TestMergeDedupsIdenticalCampaignsAndConcats(t *testing.T) {
	dir := t.TempDir()
	c := camp("01_학습자료")
	p1 := writeChunk(t, dir, "g1.json", &model.Generated{
		Campaigns: []model.Campaign{c},
		Adgroups:  []model.Adgroup{{CampaignName: c.CampaignName, AdgroupName: "그룹A"}},
		Ads:       []model.Ad{{AdName: "A_001", AdgroupName: "그룹A", Title: "제목1", Copy: "카피1"}},
	})
	p2 := writeChunk(t, dir, "g2.json", &model.Generated{
		Campaigns: []model.Campaign{c},
		Adgroups:  []model.Adgroup{{CampaignName: c.CampaignName, AdgroupName: "그룹B"}},
		Ads:       []model.Ad{{AdName: "B_001", AdgroupName: "그룹B", Title: "제목2", Copy: "카피2"}},
	})
	g, err := Merge([]string{p1, p2})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Campaigns) != 1 {
		t.Fatalf("campaigns = %d, want 1 (identical dedup)", len(g.Campaigns))
	}
	if len(g.Adgroups) != 2 || g.Adgroups[0].AdgroupName != "그룹A" || g.Adgroups[1].AdgroupName != "그룹B" {
		t.Fatalf("adgroups concat/order wrong: %+v", g.Adgroups)
	}
	if len(g.Ads) != 2 || g.Ads[1].AdName != "B_001" {
		t.Fatalf("ads concat/order wrong: %+v", g.Ads)
	}
}

func TestMergeRejectsConflictingCampaign(t *testing.T) {
	dir := t.TempDir()
	c1 := camp("01_学習資料")
	c2 := camp("01_学習資料")
	c2.BudgetMax = 99999
	p1 := writeChunk(t, dir, "g1.json", &model.Generated{Campaigns: []model.Campaign{c1}})
	p2 := writeChunk(t, dir, "g2.json", &model.Generated{Campaigns: []model.Campaign{c2}})
	_, err := Merge([]string{p1, p2})
	if err == nil || !strings.Contains(err.Error(), "충돌") {
		t.Fatalf("want conflict error, got %v", err)
	}
}

func TestMergeMissingFile(t *testing.T) {
	if _, err := Merge([]string{filepath.Join(t.TempDir(), "nope.json")}); err == nil {
		t.Fatal("want error for missing chunk")
	}
}

func TestMergeSingleInputPassesThrough(t *testing.T) {
	dir := t.TempDir()
	p := writeChunk(t, dir, "g.json", &model.Generated{Campaigns: []model.Campaign{camp("01_학습자료")}})
	g, err := Merge([]string{p})
	if err != nil || len(g.Campaigns) != 1 {
		t.Fatalf("single-input passthrough failed: %v", err)
	}
}
