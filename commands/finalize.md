---
description: 검수 완료된 워크북을 회수해 승인분만 공식 3시트 업로드 파일로 출력한다
argument-hint: <검수 완료 review.xlsx 경로>
---

검수 워크북: $ARGUMENTS

adcopy:review-workflow 스킬을 로드해 finalize 절차를 **순서 그대로** 수행하라.
같은 작업 폴더(.adcopy/)의 generated.json을 원본으로 사용한다.

핵심 가드레일:
- review-in의 Problems가 있으면 중단하고 목록을 보고한다.
- "부분 재생성"은 최종 파일에 바로 넣지 않는다 — 재생성 후 재검수를 요청한다.
- 최종 보고에 max_bid 수동 입력 안내(빈칸 유지 이유)를 반드시 포함한다.
