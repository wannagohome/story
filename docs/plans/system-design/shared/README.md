# Shared Layer (공유 계층)

모든 모듈이 공유하는 타입, 스키마, 프로토콜 정의. 서버와 클라이언트 간 **계약(contract)**의 역할.

## 문서 구조

| 문서 | 내용 |
|------|------|
| [types.md](./types.md) | 도메인 엔티티 타입 (Game, Player, World, NPC 등) |
| [events.md](./events.md) | 게임 이벤트 타입 (Go interface 패턴) |
| [protocol.md](./protocol.md) | 클라이언트-서버 WebSocket 메시지 프로토콜 |
| [schemas.md](./schemas.md) | Go struct validation (AI 출력 검증용) |

## 원칙

- **서버와 클라이언트 모두 동일한 타입을 사용한다.** 타입 불일치로 인한 런타임 오류를 방지.
- **모든 게임 이벤트는 Go interface로 정의한다.** `EventType()` 메서드로 구분, 타입 안전한 분기.
- **프로토콜 메시지도 타입별 Go struct로 정의한다.** 클라이언트→서버, 서버→클라이언트 각각.
- **AI 출력 검증은 Go struct + Validate() 메서드로.** 런타임에 구조적 검증 수행.
