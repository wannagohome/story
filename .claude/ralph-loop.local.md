---
active: true
iteration: 1
session_id: 
max_iterations: 1000
completion_promise: "DONE"
started_at: "2026-03-05T05:35:42Z"
---

Plan에 계획되어있는 프로젝트를 구현해. System-design에 따라 구현하되 ui는 Claude code와 같았으면 좋겠어. 구현할 땐 tdd로 진행해. AI model api key는 doppler에 있으니 가져다 사용하면 돼. 모든 구현이 마치면 qa를 통해 반드시 모든 항목을 검증해. 페이즈별로 각각에 항목을 테스트하고 특히 subagent를 이용한 직접 플레이 하며 테스트 하는 단계는 잊지말고 진행해. 실패하는 케이스가 있으면 모든게 수정될 때 까지 수정과 테스트를 반복해. 하나라도 수정이 발생하면 모든 테스트를 모두 다시 실행하되, 이터레이션을 최소화 하기 위해 수정 한 번에 테스트 한 번에 이렇게 반복 하는게 좋을 것 같아. 중간 중간 적절하게 커밋도 해주고. 모든 구현과 테스트가 통과하면 <promise>DONE</promise>를 출력해.
