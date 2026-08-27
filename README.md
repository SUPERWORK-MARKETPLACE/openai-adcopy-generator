# adcopy — ChatGPT 광고 대화 맥락·카피 생성 에이전트

광고주가 제공한 **상품·타깃·혜택·이미지·정책** 정보를 기반으로 다양한 **대화 맥락**을 확장하고,
**광고그룹 · Context Hints · 제목 · 카피**를 대량으로 **생성 → 검수 → 출력**하는 Claude Code
플러그인이다.

> 이것은 "엑셀 변환기"가 **아니다**. 입력 셀을 출력 셀로 매핑하는 문제가 아니라, 문맥별 광고
> 문구를 **확장·검수·출력**하는 에이전트다.

---

## 워크플로 (Human-in-the-loop)

```
광고주/대행사 ──(워크북 제공)──▶ /adcopy:generate ──▶ 검수 워크북(review.xlsx)
                                                          │
                                              운영자 검수·수정(상태값 입력)
                                                          │
                                                          ▼
                          공식 3시트 파일 ◀── /adcopy:finalize ◀── 승인
                                │
                     업로드 후 광고 시스템에서 max_bid 수동 입력
```

- **AI는 생성만 한다. 자동 업로드하지 않는다.** 운영자 검수와 광고주 확인을 거친 뒤에만 최종
  파일이 만들어진다.
- **`max_bid`는 최종 파일에서 항상 빈칸** — 값을 넣으면 업로드 오류. 업로드 후 시스템에서 수동
  입력한다.

---

## 설치 (Claude Code 플러그인)

```
/plugin marketplace add <이 저장소 URL 또는 로컬 경로>
/plugin install adcopy@nasmedia-tools
```

포함된 바이너리(`bin/`, Windows·macOS)가 함께 설치된다. 직접 빌드하려면 아래 **개발** 참조.

---

## 사용법

### 1) 생성 — `/adcopy:generate <입력 워크북.xlsx>`

광고주 통합 워크북을 입력받아 맥락을 확장하고 광고그룹·Context Hints·제목·카피를 생성한 뒤,
운영자용 **검수 워크북(`review.xlsx`)**을 만든다. 작업 산출물은 입력 파일 옆
`<입력파일>.adcopy/` 폴더에 남는다.

- 입력 검증(DRM·시트 누락·URL 접근 실패) 통과 전에는 생성을 시작하지 않는다.
- 제공된 사실 안에서만 생성하고, 정보 충돌은 **"광고주 확인 필요"**로 표시한다.

### 2) 검수 (광고주·운영자)

`review.xlsx`의 검수 시트에서 각 항목의 `검수상태`를 고른다(광고주 검수 결과 입력란):
`무수정 승인` / `수정 후 승인` / `사용 불가` / `부분 재생성`.
`검수상태`는 전 행 빈칸으로 나온다 — 자동 검수 결과(`광고주 확인 필요` 등 플래그)는
`validation_status` 열에서 확인하고, 판정은 확인 후 직접 기입한다.

### 3) 출력 — `/adcopy:finalize <검수 완료 review.xlsx>`

검수 워크북을 회수해 **승인분만** 공식 3시트 업로드 파일로 출력한다. `부분 재생성`은 바로
넣지 않고 재생성·재검수를 거친다.

---

## 산출물 — 공식 3시트

| 시트 | 컬럼 |
|---|---|
| `campaigns` | campaign_name · budget_max · budget_type · launch_date · end_date · objective · target_countries |
| `adgroups` | campaign_name · adgroup_name · **max_bid(빈칸)** · keywords(JSON 배열) |
| `ads` | adgroup_name · title · copy · link · image_link |

- `keywords`·`target_countries`는 **JSON 배열 문자열**(예: `["KR"]`).
- 내부 추적 필드(`ad_name`, `source_*`, `generation_basis`, `validation_status` 등)는 검수용으로
  별도 관리하며 업로드 파일에서는 제외된다.

주요 형식 제약: 제목 최대 24자 · 카피 최대 48자 · CTA형 종결 광고그룹당 30% 이하 ·
Context Hints 광고그룹당 5~10개 이상(검색어형 70%±10 / 질문·상황형 30%) · 이미지 1:1(640×640~1200×1200px).
자세한 규칙은 [CLAUDE.md](CLAUDE.md) 참조.

---

## 구성

### 스킬 (`skills/`)

| 스킬 | 역할 |
|---|---|
| `pipeline` | 생성 파이프라인 총괄 — 입력 검증부터 검수 워크북 출력까지 순서 고정 |
| `context-expansion` | 맥락 확장·광고그룹·Context Hints 설계, 구매 여정 퍼널 6단계 |
| `copy-rules` | 제목·카피 작성/금지 규칙(사실성·출처 우선순위·경쟁사·개인정보·민감주제) |
| `review-workflow` | 검수 엑셀 왕복과 최종 파일 출력 |

### CLI (`tool/`, Go)

플러그인이 내부적으로 호출하는 결정적 도구. 단독 실행도 가능하다.

```
adcopy version
adcopy inspect  <in.xlsx>                     # 광고주 워크북 → JSON
adcopy checkurls <urls.json>                  # URL 접근성 검사
adcopy validate <generated.json>              # 형식 검증 규칙
adcopy merge -o <merged.json> <chunk.json>... # 대량 생성 청크 병합
adcopy review-out <generated.json> -o <xlsx>  # 검수 워크북 생성
adcopy review-in  <review.xlsx> <generated.json>  # 검수 결과 회수
adcopy export    <approved.json> -o <xlsx>    # 공식 3시트 업로드 파일
```

---

## 개발

```powershell
cd tool
go test ./...                                             # 테스트
go vet ./...                                              # 린트
powershell -ExecutionPolicy Bypass -File .\build.ps1      # bin/ 3종 크로스컴파일
go run ./cmd/adcopy <command>                             # 개발용 직접 실행
```

운영 매뉴얼·도메인 규칙·제약은 [CLAUDE.md](CLAUDE.md)에 정리돼 있다.
