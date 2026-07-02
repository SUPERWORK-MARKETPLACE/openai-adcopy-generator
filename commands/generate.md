---
description: 광고주 워크북에서 광고그룹·Context Hints·제목·카피를 생성해 검수 워크북을 만든다
argument-hint: <입력 워크북.xlsx 경로>
---

입력 워크북: $ARGUMENTS

adcopy:pipeline 스킬을 로드해 그 절차를 **순서 그대로** 수행하라. 생성 단계에서는
adcopy:context-expansion과 adcopy:copy-rules 스킬도 로드해 규칙을 적용하라.

핵심 가드레일 (스킬과 중복이지만 절대 위반 금지):
- 입력 검증(DRM·시트 누락·URL 접근 실패) 통과 전에는 생성을 시작하지 않는다.
- 제공된 사실 안에서만 생성하고, 정보 충돌은 "광고주 확인 필요"로 표시한다.
- max_bid는 어디에도 쓰지 않는다.
- 마지막에 검수 워크북 경로와 다음 단계(/adcopy:finalize)를 운영자에게 안내한다.
