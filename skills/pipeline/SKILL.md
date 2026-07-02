---
name: pipeline
description: 광고 카피 생성 파이프라인 총괄 — 입력 검증부터 검수 워크북 출력까지 순서 고정 실행. /adcopy:generate가 로드한다.
---

# 생성 파이프라인 (순서 고정 — 건너뛰기 금지)

## 0. 바이너리 선택

adcopy CLI 경로 (OS에 맞게 택1):
- Windows: `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-windows-amd64.exe`
- macOS(Apple Silicon): `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-darwin-arm64`
- macOS(Intel): `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-darwin-amd64`

이하 `adcopy`로 표기. 종료 코드: 0 정상 / 1 검증 위반 발견 / 2 실행 오류.

## 1. 작업 폴더

입력 워크북과 같은 위치에 `<파일명(확장자 제외)>.adcopy/` 폴더를 만들고 모든 중간 산출물을 저장:
`workbook.json`(입력 덤프) → `urls.json`·`urlcheck.json` → `generated.json` → `validate-report.json` → `review.xlsx`. 세션이 끊겨도 이 파일들로 재개한다.

## 2. 입력 검증 (생성 시작 전 — 실패 시 부분 진행 없이 중단·보고)

1. `adcopy inspect <입력.xlsx>` → `workbook.json` 저장. DRM/암호화 오류면 "표준 xlsx로 다시 저장해 주세요"라고 안내하고 중단.
2. 덤프에서 시트·필드 파악: campaigns(필수), 상품·브리프, 이미지소재, 고객질문·검색데이터, 주력 키워드, 정책·참고자료. campaigns 또는 브리프(상품·대표 랜딩 URL)가 없으면 중단·보고.
3. 모든 랜딩·이미지 URL을 모아 `urls.json`(`{"urls": [...]}`)으로 저장 → `adcopy checkurls urls.json`. 실패 URL이 있으면 **생성 전에** 목록 보고 후 중단 (운영자가 계속 진행을 명시적으로 지시한 경우에만 해당 소재 제외하고 진행).
4. 프로모션 종료일이 지난 소재는 생성 대상에서 제외하고 제외 사유에 기록.

## 3. 조건 설정 (운영자 대화)

AI 추천 기본값을 제시하고 운영자 조정을 받는다: 포함/제외 퍼널 단계(기본: ①~⑤ 포함, ⑥ 사용·문제 해결은 제외), 단계별 비중, 광고그룹 수, 광고그룹당 카피 수(기본 2~3), 검수 엄격도. 운영자가 무응답이면 기본값으로 진행하되 보고에 명시.

## 4. 생성 (adcopy:context-expansion, adcopy:copy-rules 스킬 로드 후)

- 맥락 확장 → 광고그룹 → Context Hints → 제목·카피 → URL 연결 순. 상품 정보를 바로 카피로 바꾸지 말 것.
- **광고그룹 단위로 생성해 `generated.json`에 누적 저장** (대량 생성 시 컨텍스트 관리). 규모가 크면 광고그룹별 서브에이전트로 분산.
- 소재별 랜딩 URL이 있으면 `link`에 우선 적용, 없으면 대표 랜딩 URL.
- 브리프·랜딩·이미지 간 정보가 충돌하는 항목은 생성하지 말고 `validation_status: "광고주 확인 필요"`로 표시.

## 5. generated.json 스키마 (CLI 계약 — 필드명·타입 엄수)

```json
{
  "campaigns": [{
    "campaign_name": "01_학습자료", "budget_max": 25000, "budget_type": "daily",
    "launch_date": "2026-07-01", "end_date": "2026-07-31",
    "objective": "Views", "target_countries": ["KR"]
  }],
  "adgroups": [{
    "campaign_name": "01_학습자료", "adgroup_name": "01_훈련앱",
    "keywords": [{"text": "초등 영어 앱 추천", "origin": "customer_data"}],
    "trace": {"source_type": "브리프", "source_url": "", "source_excerpt": "",
      "generation_basis": "SKU=...; Persona=...; 문제=...; 상황=...; 퍼널=...; 세부의도=...; 메시지=...",
      "confidence_score": 0.9, "validation_status": "", "review_comment": "", "exclusion_reason": ""}
  }],
  "ads": [{
    "ad_name": "KID_01_001", "adgroup_name": "01_훈련앱",
    "title": "...", "copy": "...", "link": "https://...", "image_link": "https://...",
    "trace": { "...": "adgroups와 동일 구조" }
  }]
}
```

- `keywords[].origin`: 실제 고객 데이터 기반이면 `customer_data`, AI 추론이면 `ai_inferred` (결과에서 구분 표시 의무).
- **`max_bid`는 절대 쓰지 않는다** — 값이 있으면 validate가 오류로 잡는다.
- `campaigns`는 입력 워크북 값을 그대로 옮긴다(AI가 지어내지 않음).
- `ad_name` 형식: `캠페인/SKU 코드 + 광고그룹 코드 + 크리에이티브 순번` (예: KID_01_001). 내부 관리용 — 업로드 파일에는 포함되지 않는다.

## 6. 자동 검수 루프

1. `adcopy validate generated.json` → `validate-report.json`.
2. errors가 있으면 해당 항목만 재생성/수정 후 재실행 (최대 3회 반복, 그래도 남으면 해당 항목 제외 + 제외 사유 기록).
3. 형식 검증과 별개로 의미 검수(adcopy:copy-rules의 금지 규칙)를 스스로 점검.

## 7. 검수 워크북 출력·보고

1. `adcopy review-out generated.json -o review.xlsx`
2. 운영자에게 보고: 생성 수량(캠페인/광고그룹/광고), 제외 항목과 사유, `광고주 확인 필요` 건수, 자동 검수 결과 요약, review.xlsx 경로와 다음 단계(엑셀 검수 → `/adcopy:finalize`).
3. 자동 검수는 **오류 후보 선별일 뿐** — 최종 판단은 운영자·광고주가 한다는 점을 함께 안내.
