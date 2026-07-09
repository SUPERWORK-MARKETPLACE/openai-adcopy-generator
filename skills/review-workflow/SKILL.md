---
name: review-workflow
description: 검수 엑셀 왕복과 최종 파일 출력 절차 — 상태값 처리, 부분 재생성, max_bid 규칙. /adcopy:finalize가 로드한다.
---

# 검수 왕복 → 최종 출력

## 0. 바이너리 선택

adcopy CLI 경로 (OS에 맞게 택1):
- Windows: `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-windows-amd64.exe`
- macOS(Apple Silicon): `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-darwin-arm64`
- macOS(Intel): `${CLAUDE_PLUGIN_ROOT}/bin/adcopy-darwin-amd64`

이하 `adcopy`로 표기. 종료 코드: 0 정상 / 1 문제 발견 / 2 실행 오류.

## review.xlsx 필드 역할 (혼동 금지)

- `validation_status`: **자동 검수 결과(읽기 전용)** — 형식·정책·사실성 검사 결과와 문제 유형, AI 검수 노트. 운영자가 수정하지 않는다.
- `검수상태`: **운영자 판정 드롭다운** — 아래 5종. finalize는 이 열만 본다(review-in JSON의 `review_status`).
- `review_comment`: 운영자·광고주 수정 의견·재생성 사유 기입란.

## 검수 상태값 (운영자가 review.xlsx `검수상태` 드롭다운에서 선택)

| 상태 | finalize 처리 |
|---|---|
| 무수정 승인 | 그대로 최종 파일에 포함 |
| 수정 후 승인 | 운영자가 시트에서 고친 title/copy/keywords를 반영해 포함 (재검증 필수) |
| 사용 불가 | 제외 |
| 부분 재생성 | review_comment를 반영해 재생성 → 재검증 → 새 review.xlsx로 재검수 요청 (최종 파일에 바로 넣지 않는다) |
| 광고주 확인 필요 | 제외하고 확인 대기 목록으로 보고 |
| (빈칸) | 미검수 — 제외하고 미검수 목록으로 보고 |

## finalize 절차 (순서 고정)

1. `adcopy review-in review.xlsx generated.json` — Problems가 있으면 **중단**하고 목록 보고 (원본과 매칭 안 되는 행·허용 외 검수상태 값 등). 운영자 판정은 결과 JSON의 `review_status` 필드로 들어온다.
2. 상태별 처리(위 표). `수정 후 승인` 반영분과 `부분 재생성` 결과물은 다시 `adcopy validate`를 통과해야 한다.
   - 운영자가 keywords를 수정한 경우: 기존 텍스트와 일치하는 힌트는 원래 origin을 유지하고,
     운영자가 새로 추가한 힌트는 `customer_data`(사람 제공)로 기록한다.
3. 승인분만 모아 `approved.json` 작성: campaigns 전체 + 승인 광고가 하나 이상 남은 adgroups + 승인 ads. 승인 광고가 0건인 광고그룹은 제외(고아 그룹 방지).
4. `adcopy export approved.json -o final.xlsx` (export가 내부적으로 재검증 — 오류 시 중단).
5. 완료 보고에 반드시 포함:
   - 시트별 건수, 제외·대기 목록(사유 포함)
   - **max_bid는 빈칸입니다 — 값을 넣으면 업로드 오류가 나므로, 업로드 후 광고 시스템에서 직접 입력하세요**
   - 이 파일은 업로드용 초안이며 최종 확인·업로드는 광고주/운영자 몫 (자동 업로드 없음)

## 절대 규칙

- 자동 검수는 오류 후보 선별일 뿐 — 승인 권한은 운영자·광고주에게 있다. AI가 `검수상태`(운영자 판정)를 임의로 채우지 마라. `validation_status`(자동 검수 결과)에 "광고주 확인 필요"를 표시하는 것은 AI 몫이다.
- `max_bid`에 어떤 값도 쓰지 마라.
- 근거 추적 필드는 review.xlsx까지만 — final.xlsx에는 절대 포함하지 않는다 (export가 보장).
