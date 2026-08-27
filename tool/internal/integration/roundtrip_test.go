package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"adcopy/internal/export"
	"adcopy/internal/model"
	"adcopy/internal/review"
	"adcopy/internal/validate"
)

// fullGenerated builds a realistic multi-axis KB국민카드 fixture (all products
// fictional). It demonstrates the core promise of the agent: a handful of input
// axes (SKU × persona × 구매 여정 단계) explode into many adgroups and many
// title/copy sets. It also exercises the policy validator — a global
// banned-term list, fully avoided so the fixture validates clean.
func fullGenerated() *model.Generated {
	// kw builds a keyword slice with a shared origin.
	kw := func(origin string, texts ...string) []model.Keyword {
		out := make([]model.Keyword, 0, len(texts))
		for _, t := range texts {
			out = append(out, model.Keyword{Text: t, Origin: origin})
		}
		return out
	}
	const (
		life   = "https://card.kbcard.com/sample/life"
		online = "https://card.kbcard.com/sample/online"
		travel = "https://card.kbcard.com/sample/travel"
		imgL   = "https://img.kbcard.com/sample/life.png"
		imgO   = "https://img.kbcard.com/sample/online.png"
		imgT   = "https://img.kbcard.com/sample/travel.png"
		camp   = "kbcard_lifestyle_2026"
	)

	return &model.Generated{
		Campaigns: []model.Campaign{{
			CampaignName: camp, BudgetMax: 30000, BudgetType: "daily",
			LaunchDate: "2026-07-01", EndDate: "2026-07-31",
			Objective: "Traffic", TargetCountries: []string{"KR"},
		}},

		// Global banned expressions (input 정책 시트 공통 금지 표현).
		Policy: &model.Policy{BannedTerms: []string{
			"무조건 발급", "누구나", "최고", "반드시 절약", "혜택 보장", "업계 1위",
		}},

		// 7 adgroups: 3 SKU × personas × funnel stages (②제품발견/③비교검토/⑤신청전환).
		// 각 그룹의 검색어형 힌트는 70%±10(60~80%) — S2 keyword_searchform_ratio 통과형.
		Adgroups: []model.Adgroup{
			// SKU 생활비 혜택 카드
			{CampaignName: camp, AdgroupName: "life_직장인_교통비_비교검토",
				Keywords: kw("customer_data", "교통비 카드 비교", "직장인 생활비 카드", "직장인 통신 요금 할인 카드", "편의점 정기구독 할인 카드 비교", "편의점 지출이 부담될 때", "생활비 카드 어떻게 고를까")},
			{CampaignName: camp, AdgroupName: "life_자취생_생활비_제품발견",
				Keywords: kw("ai_inferred", "자취 생활비 절약", "1인가구 카드 추천", "자취생 편의점 할인 카드", "1인가구 통신비 절약 카드", "혼자 살면 생활비가 얼마나 들까", "자취 시작하면 카드 뭐가 필요할까")},
			{CampaignName: camp, AdgroupName: "life_직장인_정기구독_신청전환",
				Keywords: kw("ai_inferred", "정기구독 혜택 카드", "구독 서비스 요금이 부담될 때", "정기결제 할인 카드 신청", "직장인 정기구독 카드", "정기결제 혜택은 어떻게 챙길까")},
			// SKU 온라인 쇼핑 혜택 카드
			{CampaignName: camp, AdgroupName: "online_맞벌이_온라인쇼핑_비교검토",
				Keywords: kw("ai_inferred", "온라인 쇼핑 카드 비교", "맞벌이 온라인 결제 할인 카드", "구독 결제가 많은데 카드 바꿔야 할까", "맞벌이 카드 추천", "온라인 쇼핑 지출이 늘어날 때")},
			{CampaignName: camp, AdgroupName: "online_사회초년생_구독결제_신청전환",
				Keywords: kw("ai_inferred", "구독 결제 카드 신청", "사회초년생 구독 할인 카드", "사회초년생 첫 카드", "구독 할인 혜택은 어떻게 신청할까", "카드 신청 전에 뭘 확인해야 할까")},
			// SKU 여행·해외결제 혜택 카드
			{CampaignName: camp, AdgroupName: "travel_해외여행객_해외결제_제품발견",
				Keywords: kw("customer_data", "해외여행 갈 때 카드 어떻게 준비할까", "항공 마일리지 카드", "해외 결제 수수료 할인 카드", "해외여행 카드 추천", "해외 결제 수수료 얼마나 나올까")},
			{CampaignName: camp, AdgroupName: "travel_해외여행객_항공숙박_신청전환",
				Keywords: kw("ai_inferred", "해외 결제 카드 신청", "항공권 할인 카드", "호텔 예약 할인 카드 신청", "출국 전에 카드 준비 뭐가 필요할까", "해외에서 카드 쓸 때 수수료 아낄 수 있을까")},
		},

		// 13 ads across the 7 groups; distinct personas/stages → distinct copy.
		Ads: []model.Ad{
			// life_직장인_교통비_비교검토
			{AdName: "KB_LA01", AdgroupName: "life_직장인_교통비_비교검토",
				Title: "교통비 카드 혜택 비교해 볼까요",
				Copy:  "교통·통신·편의점 혜택을 한눈에 비교하고 골라 보세요",
				Link:  life, ImageLink: imgL},
			{AdName: "KB_LA02", AdgroupName: "life_직장인_교통비_비교검토",
				Title: "생활비 카드 비교가 처음이라면",
				Copy:  "정기구독과 통신 요금까지 챙기는 혜택을 비교해 보세요",
				Link:  life, ImageLink: imgL},
			// life_자취생_생활비_제품발견
			{AdName: "KB_LB01", AdgroupName: "life_자취생_생활비_제품발견",
				Title: "자취 생활비 아끼는 카드 찾기",
				Copy:  "편의점과 통신 요금 혜택으로 자취 고정비를 줄여요",
				Link:  life, ImageLink: imgL},
			{AdName: "KB_LB02", AdgroupName: "life_자취생_생활비_제품발견",
				Title: "1인가구 생활비 절약 카드",
				Copy:  "통신 요금과 편의점 혜택으로 매달 생활비를 아껴 보세요",
				Link:  life, ImageLink: imgL},
			// life_직장인_정기구독_신청전환
			{AdName: "KB_LC01", AdgroupName: "life_직장인_정기구독_신청전환",
				Title: "정기구독 혜택 카드 신청 전에",
				Copy:  "정기구독과 통신 혜택의 이용 조건을 확인하고 신청해요",
				Link:  life, ImageLink: imgL},
			// online_맞벌이_온라인쇼핑_비교검토
			{AdName: "KB_OA01", AdgroupName: "online_맞벌이_온라인쇼핑_비교검토",
				Title: "온라인 쇼핑 카드 혜택 비교",
				Copy:  "온라인 결제와 구독 혜택을 꼼꼼히 비교하고 선택하세요",
				Link:  online, ImageLink: imgO},
			{AdName: "KB_OA02", AdgroupName: "online_맞벌이_온라인쇼핑_비교검토",
				Title: "맞벌이 온라인 결제 혜택 정리",
				Copy:  "매달 쓰는 온라인 결제와 구독을 한 카드로 모아 보세요",
				Link:  online, ImageLink: imgO},
			// online_사회초년생_구독결제_신청전환
			{AdName: "KB_OB01", AdgroupName: "online_사회초년생_구독결제_신청전환",
				Title: "첫 카드로 구독 결제 시작하기",
				Copy:  "구독 결제 혜택과 주요 이용 조건을 확인하고 신청하세요",
				Link:  online, ImageLink: imgO},
			{AdName: "KB_OB02", AdgroupName: "online_사회초년생_구독결제_신청전환",
				Title: "사회초년생 구독 카드 신청 전에",
				Copy:  "연회비와 전월 실적 등 이용 조건을 확인하고 신청해요",
				Link:  online, ImageLink: imgO},
			// travel_해외여행객_해외결제_제품발견
			{AdName: "KB_TA01", AdgroupName: "travel_해외여행객_해외결제_제품발견",
				Title: "해외여행 결제 카드 찾는다면",
				Copy:  "항공·숙박·해외 결제 혜택을 한 장으로 챙겨 보세요",
				Link:  travel, ImageLink: imgT},
			{AdName: "KB_TA02", AdgroupName: "travel_해외여행객_해외결제_제품발견",
				Title: "해외 결제 수수료 부담된다면",
				Copy:  "항공과 숙박 결제까지 해외 혜택을 한 카드로 준비해요",
				Link:  travel, ImageLink: imgT},
			// travel_해외여행객_항공숙박_신청전환
			{AdName: "KB_TB01", AdgroupName: "travel_해외여행객_항공숙박_신청전환",
				Title: "해외 결제 카드 신청 전 확인",
				Copy:  "해외 이용 수수료 등 조건을 확인하고 신청하세요",
				Link:  travel, ImageLink: imgT},
			{AdName: "KB_TB02", AdgroupName: "travel_해외여행객_항공숙박_신청전환",
				Title: "항공·숙박 혜택 카드 신청하기",
				Copy:  "항공과 숙박 혜택 조건을 확인하고 신청해 보세요",
				Link:  travel, ImageLink: imgT},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	g := fullGenerated()

	// 1. generation output must validate clean (incl. new policy checks).
	if rep := validate.Validate(g); !rep.OK {
		t.Fatalf("precondition: %+v", rep.Errors)
	}

	// 2. write review workbook (merging the validation report per row).
	reviewPath := filepath.Join(dir, "review.xlsx")
	if err := review.WriteReview(g, validate.Validate(g), reviewPath); err != nil {
		t.Fatal(err)
	}

	// 3. operator fills EVERY status cell (no blanks) — reject exactly one ad,
	//    approve every adgroup. Reject an ad from a 2-ad group so its group
	//    still has an approved survivor.
	const rejectedAd = "KB_LA01"
	f, err := excelize.OpenFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	// ads_검수 검수상태 is column J (10th header); rows start at 2.
	for i, ad := range g.Ads {
		status := model.StatusApproved
		if ad.AdName == rejectedAd {
			status = model.StatusRejected
		}
		f.SetCellValue("ads_검수", fmt.Sprintf("J%d", i+2), status)
	}
	// adgroups_검수 검수상태 is column F (6th header).
	for i := range g.Adgroups {
		f.SetCellValue("adgroups_검수", fmt.Sprintf("F%d", i+2), model.StatusApproved)
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// 4. read back — no structural problems expected.
	res, err := review.ReadReview(reviewPath, g)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) != 0 {
		t.Fatalf("problems: %v", res.Problems)
	}

	// 5. finalize: keep approved ads, drop unused adgroups (mirrors CLI logic).
	statusByAd := map[string]string{}
	for _, a := range res.Ads {
		statusByAd[a.AdName] = a.ReviewStatus
	}
	approved := &model.Generated{Campaigns: g.Campaigns, Policy: g.Policy}
	usedAdgroups := map[string]bool{}
	for _, ad := range g.Ads {
		s := statusByAd[ad.AdName]
		if s == model.StatusApproved || s == model.StatusApprovedEdited {
			approved.Ads = append(approved.Ads, ad)
			usedAdgroups[ad.AdgroupName] = true
		}
	}
	for _, ag := range g.Adgroups {
		if usedAdgroups[ag.AdgroupName] {
			approved.Adgroups = append(approved.Adgroups, ag)
		}
	}

	// 6. approved subset must validate and export.
	if rep := validate.Validate(approved); !rep.OK {
		t.Fatalf("approved subset invalid: %+v", rep.Errors)
	}
	finalPath := filepath.Join(dir, "final.xlsx")
	if err := export.Export(approved, finalPath); err != nil {
		t.Fatal(err)
	}

	out, err := excelize.OpenFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	// Robust assertions computed from the fixture, not hardcoded.
	wantApproved := len(g.Ads) - 1 // exactly one ad rejected
	ads, _ := out.GetRows("ads")
	if len(ads)-1 != wantApproved {
		t.Fatalf("ads rows = %d, want %d (header + %d approved)", len(ads), wantApproved+1, wantApproved)
	}
	ags, _ := out.GetRows("adgroups")
	if len(ags)-1 != len(usedAdgroups) {
		t.Fatalf("adgroups rows = %d, want %d (header + %d used groups)", len(ags), len(usedAdgroups)+1, len(usedAdgroups))
	}

	// The rejected ad's creative must not survive in the export.
	for _, row := range ads[1:] {
		if len(row) > 1 && row[1] == "교통비 카드 혜택 비교해 볼까요" {
			// KB_LA01's title — allowed only if a different approved ad reused it,
			// but titles are unique, so its presence means the reject leaked.
			t.Fatalf("rejected ad %s leaked into export: %v", rejectedAd, row)
		}
	}
}
