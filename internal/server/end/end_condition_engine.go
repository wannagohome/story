package end

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/story/internal/server/aiface"
	"github.com/anthropics/story/internal/server/eventbus"
	"github.com/anthropics/story/internal/server/game"
	"github.com/anthropics/story/internal/shared/events"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// SessionNotifier is the interface that SessionManager implements.
// It breaks the circular dependency between EndConditionEngine and SessionManager.
type SessionNotifier interface {
	StartEnding()
	FinishGame()
}

// ActiveVote represents an ongoing vote.
type ActiveVote struct {
	Reason      string
	Candidates  []string
	Votes       map[string]string // voterId -> candidateId
	TotalVoters int
	TimeoutMs   int64
	Timer       *time.Timer
}

// ActiveConsensus represents an ongoing consensus (solve) round.
type ActiveConsensus struct {
	Answers      map[string]string // playerId -> answer
	TotalPlayers int
	TimeoutMs    int64
	Timer        *time.Timer
}

// EndProposal represents a player-initiated end proposal.
type EndProposal struct {
	ProposerID  string
	Responses   map[string]bool // playerId -> agree/disagree
	TotalVoters int
	Timer       *time.Timer
}

// EndConditionEngine evaluates end conditions, manages votes/consensus,
// handles timeout, and triggers endings.
type EndConditionEngine struct {
	endConditions   []types.EndCondition
	timeout         *time.Timer
	warningTimers   []*time.Timer
	activeVote      *ActiveVote
	activeConsensus *ActiveConsensus
	endProposal     *EndProposal
	gameState       *game.GameStateManager
	aiLayer         aiface.AILayer
	eventBus        *eventbus.EventBus
	session         SessionNotifier
	monitoring      bool
	mu              sync.Mutex
}

// NewEndConditionEngine creates a new EndConditionEngine.
func NewEndConditionEngine(
	gs *game.GameStateManager,
	ail aiface.AILayer,
	bus *eventbus.EventBus,
	sn SessionNotifier,
) *EndConditionEngine {
	return &EndConditionEngine{
		gameState: gs,
		aiLayer:   ail,
		eventBus:  bus,
		session:   sn,
	}
}

// StartMonitoring begins end condition monitoring with the given conditions and timeout.
func (ece *EndConditionEngine) StartMonitoring(endConditions []types.EndCondition, timeoutMinutes int) {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	ece.endConditions = endConditions
	ece.monitoring = true

	// Subscribe to game events for condition checking
	ch := ece.eventBus.SubscribeGameEvent()
	go ece.listenGameEvents(ch)

	// Start timeout management
	ece.startTimeout(timeoutMinutes)
}

// StopMonitoring stops all monitoring, timers, and votes.
func (ece *EndConditionEngine) StopMonitoring() {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	ece.monitoring = false

	if ece.timeout != nil {
		ece.timeout.Stop()
		ece.timeout = nil
	}

	for _, t := range ece.warningTimers {
		t.Stop()
	}
	ece.warningTimers = nil

	if ece.activeVote != nil && ece.activeVote.Timer != nil {
		ece.activeVote.Timer.Stop()
	}
	ece.activeVote = nil

	if ece.activeConsensus != nil && ece.activeConsensus.Timer != nil {
		ece.activeConsensus.Timer.Stop()
	}
	ece.activeConsensus = nil

	if ece.endProposal != nil && ece.endProposal.Timer != nil {
		ece.endProposal.Timer.Stop()
	}
	ece.endProposal = nil
}

// StartVote initiates a vote among all players.
func (ece *EndConditionEngine) StartVote(reason string, candidates []string, timeoutSeconds int) {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	totalVoters := len(ece.gameState.GetAllPlayerIDs())
	vote := &ActiveVote{
		Reason:      reason,
		Candidates:  candidates,
		Votes:       make(map[string]string),
		TotalVoters: totalVoters,
		TimeoutMs:   int64(timeoutSeconds) * 1000,
	}

	vote.Timer = time.AfterFunc(time.Duration(timeoutSeconds)*time.Second, func() {
		ece.finalizeVote()
	})

	ece.activeVote = vote

	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "vote_started",
		Data: events.GameEndEventData{
			Reason: reason,
		},
	})
}

// CastVote records a player's vote.
func (ece *EndConditionEngine) CastVote(playerID string, targetID string) error {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	if ece.activeVote == nil {
		return ErrNoActiveVote
	}

	ece.activeVote.Votes[playerID] = targetID

	// Publish progress
	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "vote_progress",
		Data: events.GameEndEventData{
			Reason: ece.activeVote.Reason,
		},
	})

	// Check if all votes are in
	if len(ece.activeVote.Votes) >= ece.activeVote.TotalVoters {
		go ece.finalizeVote()
	}

	return nil
}

// SubmitSolution records a player's solution answer.
func (ece *EndConditionEngine) SubmitSolution(playerID string, answer string) error {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	if ece.activeConsensus == nil {
		return ErrNoActiveConsensus
	}

	ece.activeConsensus.Answers[playerID] = answer

	// Publish progress
	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "solve_progress",
		Data:      events.GameEndEventData{},
	})

	// Check if all answers are in
	if len(ece.activeConsensus.Answers) >= ece.activeConsensus.TotalPlayers {
		go ece.finalizeConsensus()
	}

	return nil
}

// ProposeEnd initiates a player-proposed end vote.
func (ece *EndConditionEngine) ProposeEnd(proposerID string) error {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	if ece.endProposal != nil {
		return ErrEndVoteAlreadyOpen
	}

	totalVoters := len(ece.gameState.GetAllPlayerIDs())
	proposal := &EndProposal{
		ProposerID:  proposerID,
		Responses:   make(map[string]bool),
		TotalVoters: totalVoters,
	}
	// Proposer auto-agrees
	proposal.Responses[proposerID] = true

	proposal.Timer = time.AfterFunc(60*time.Second, func() {
		ece.finalizeEndProposal()
	})

	ece.endProposal = proposal

	proposer := ece.gameState.GetPlayer(proposerID)
	proposerName := ""
	if proposer != nil {
		proposerName = proposer.Nickname
	}

	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "end_proposed",
		Data: events.GameEndEventData{
			Reason: proposerName + " proposed to end the game",
		},
	})

	// Check if only 1 player (auto-pass)
	if totalVoters <= 1 {
		go ece.finalizeEndProposal()
	}

	return nil
}

// RespondToEndProposal records a player's response to an end proposal.
func (ece *EndConditionEngine) RespondToEndProposal(playerID string, agree bool) error {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	if ece.endProposal == nil {
		return ErrNoEndProposal
	}

	ece.endProposal.Responses[playerID] = agree

	// Check if all responses are in
	if len(ece.endProposal.Responses) >= ece.endProposal.TotalVoters {
		go ece.finalizeEndProposal()
	}

	return nil
}

// listenGameEvents listens for game events and checks end conditions.
func (ece *EndConditionEngine) listenGameEvents(ch <-chan types.GameEvent) {
	for event := range ch {
		ece.mu.Lock()
		if !ece.monitoring {
			ece.mu.Unlock()
			return
		}
		ece.mu.Unlock()

		// Only check on significant events
		eventType := event.EventType()
		switch eventType {
		case "vote_ended", "story_event", "clue_found", "action_result":
			ece.checkEndConditions(event)
		}
	}
}

// checkEndConditions evaluates all end conditions against the current game state.
func (ece *EndConditionEngine) checkEndConditions(event types.GameEvent) {
	ece.mu.Lock()
	defer ece.mu.Unlock()

	if !ece.monitoring {
		return
	}

	for _, condition := range ece.endConditions {
		switch condition.TriggerType {
		case "timeout":
			// Handled by startTimeout timer
			continue
		case "vote":
			// Evaluated when vote ends
			continue
		case "consensus":
			// Evaluated when consensus completes
			continue
		case "event":
			if ece.matchEventCondition(event, condition) {
				go ece.triggerEnding(condition.Description)
				return
			}
		case "ai_judgment":
			go func(cond types.EndCondition) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				gameCtx := ece.buildGameContext()
				met, err := ece.aiLayer.JudgeEndCondition(ctx, gameCtx, cond)
				if err != nil {
					slog.Warn("AI end condition judgment failed", "error", err)
					return
				}
				if met {
					ece.triggerEnding(cond.Description)
				}
			}(condition)
		}
	}
}

// matchEventCondition checks if a game event matches an event-based end condition.
func (ece *EndConditionEngine) matchEventCondition(event types.GameEvent, condition types.EndCondition) bool {
	criteria := condition.TriggerCriteria
	if criteria == nil {
		return false
	}

	eventTypeStr, ok := criteria["eventType"].(string)
	if !ok {
		return false
	}

	return event.EventType() == eventTypeStr
}

// triggerEnding stops monitoring and triggers the ending sequence.
func (ece *EndConditionEngine) triggerEnding(reason string) {
	ece.mu.Lock()
	if !ece.monitoring {
		ece.mu.Unlock()
		return
	}
	ece.monitoring = false
	ece.mu.Unlock()

	ece.StopMonitoring()
	ece.session.StartEnding()

	go func() {
		gameCtx := ece.buildGameContext()

		var ending *schemas.Ending
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(1<<uint(attempt-1)) * time.Second
				time.Sleep(backoff)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			ending, err = ece.aiLayer.GenerateEndings(ctx, gameCtx, reason)
			cancel()
			if err == nil {
				break
			}
			slog.Warn("ending generation failed, retrying", "attempt", attempt+1, "error", err)
		}

		var endData types.GameEndData
		if err != nil {
			slog.Error("ending generation failed after all retries, using fallback", "error", err)
			endData = ece.buildFallbackEnding(reason)
		} else {
			endData = ece.convertEndingToGameEndData(ending)
		}

		endData.SecretReveal = ece.buildSecretRevealFromGameState()
		ece.eventBus.PublishSendEndings(endData)

		ece.session.FinishGame()
	}()
}

// startTimeout sets up timeout and warning timers.
func (ece *EndConditionEngine) startTimeout(minutes int) {
	// 5 minutes warning (skip if game is 5 min or less)
	if minutes > 5 {
		t := time.AfterFunc(time.Duration(minutes-5)*time.Minute, func() {
			ece.eventBus.PublishGameEvent(events.TimeWarningEvent{
				BaseEvent: newBaseEvent("all", "", nil),
				Type:      "time_warning",
				Data:      events.TimeWarningData{RemainingMinutes: 5},
			})
		})
		ece.warningTimers = append(ece.warningTimers, t)
	}

	// 1 minute warning
	if minutes > 1 {
		t := time.AfterFunc(time.Duration(minutes-1)*time.Minute, func() {
			ece.eventBus.PublishGameEvent(events.TimeWarningEvent{
				BaseEvent: newBaseEvent("all", "", nil),
				Type:      "time_warning",
				Data:      events.TimeWarningData{RemainingMinutes: 1},
			})
		})
		ece.warningTimers = append(ece.warningTimers, t)
	}

	// Timeout
	ece.timeout = time.AfterFunc(time.Duration(minutes)*time.Minute, func() {
		ece.triggerEnding("timeout")
	})
}

// finalizeVote tallies votes and publishes results.
func (ece *EndConditionEngine) finalizeVote() {
	ece.mu.Lock()
	vote := ece.activeVote
	if vote == nil {
		ece.mu.Unlock()
		return
	}
	if vote.Timer != nil {
		vote.Timer.Stop()
	}
	ece.activeVote = nil
	ece.mu.Unlock()

	// Tally votes
	tally := make(map[string]int)
	for _, candidateID := range vote.Votes {
		tally[candidateID]++
	}

	// Find winner
	var winnerID string
	maxVotes := 0
	for id, count := range tally {
		if count > maxVotes {
			maxVotes = count
			winnerID = id
		}
	}

	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "vote_ended",
		Data: events.GameEndEventData{
			Reason:       vote.Reason,
			CommonResult: winnerID,
		},
	})
}

// finalizeConsensus evaluates consensus answers.
func (ece *EndConditionEngine) finalizeConsensus() {
	ece.mu.Lock()
	consensus := ece.activeConsensus
	if consensus == nil {
		ece.mu.Unlock()
		return
	}
	if consensus.Timer != nil {
		consensus.Timer.Stop()
	}
	ece.activeConsensus = nil
	ece.mu.Unlock()

	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "solve_result",
		Data: events.GameEndEventData{
			Reason: "consensus",
		},
	})
}

// finalizeEndProposal tallies end proposal responses.
func (ece *EndConditionEngine) finalizeEndProposal() {
	ece.mu.Lock()
	proposal := ece.endProposal
	if proposal == nil {
		ece.mu.Unlock()
		return
	}
	if proposal.Timer != nil {
		proposal.Timer.Stop()
	}
	ece.endProposal = nil

	agreed := 0
	disagreed := 0
	for _, agree := range proposal.Responses {
		if agree {
			agreed++
		} else {
			disagreed++
		}
	}
	// Non-responses count as disagree
	disagreed += proposal.TotalVoters - len(proposal.Responses)
	passed := agreed > proposal.TotalVoters/2
	ece.mu.Unlock()

	ece.eventBus.PublishGameEvent(events.GameEndEvent{
		BaseEvent: newBaseEvent("all", "", nil),
		Type:      "end_vote_result",
		Data: events.GameEndEventData{
			Reason: "player_proposed",
		},
	})

	if passed {
		ece.triggerEnding("player_proposed")
	}
}

// buildGameContext constructs the GameContext for AI calls.
func (ece *EndConditionEngine) buildGameContext() types.GameContext {
	return types.GameContext{
		World:        ece.gameState.GetWorld(),
		CurrentState: ece.gameState.GetFullState(),
	}
}

// buildFallbackEnding creates a hardcoded fallback ending when AI generation fails.
func (ece *EndConditionEngine) buildFallbackEnding(_ string) types.GameEndData {
	commonResult := "The game has ended. All players did their best."
	playerIDs := ece.gameState.GetAllPlayerIDs()
	endings := make([]types.PlayerEnding, len(playerIDs))
	for i, id := range playerIDs {
		endings[i] = types.PlayerEnding{
			PlayerID:    id,
			Summary:     "Game Over",
			GoalResults: []types.GoalResult{},
			Narrative:   "The story has come to a close.",
		}
	}
	return types.GameEndData{
		CommonResult:  commonResult,
		PlayerEndings: endings,
	}
}

// convertEndingToGameEndData converts an AI Ending schema to GameEndData.
func (ece *EndConditionEngine) convertEndingToGameEndData(ending *schemas.Ending) types.GameEndData {
	playerEndings := make([]types.PlayerEnding, len(ending.PlayerEndings))
	for i, pe := range ending.PlayerEndings {
		goalResults := make([]types.GoalResult, len(pe.GoalResults))
		for j, gr := range pe.GoalResults {
			goalResults[j] = types.GoalResult{
				GoalID:      gr.GoalID,
				Description: gr.Description,
				Achieved:    gr.Achieved,
				Evaluation:  gr.Evaluation,
			}
		}
		playerEndings[i] = types.PlayerEnding{
			PlayerID:    pe.PlayerID,
			Summary:     pe.Summary,
			GoalResults: goalResults,
			Narrative:   pe.Narrative,
		}
	}
	return types.GameEndData{
		CommonResult:  ending.CommonResult,
		PlayerEndings: playerEndings,
	}
}

// buildSecretRevealFromGameState constructs the SecretReveal from the current game state.
func (ece *EndConditionEngine) buildSecretRevealFromGameState() types.SecretReveal {
	world := ece.gameState.GetWorld()
	state := ece.gameState.GetFullState()

	// Player secrets
	var playerSecrets []types.PlayerSecretEntry
	for _, player := range state.Players {
		if player.Role != nil {
			playerSecrets = append(playerSecrets, types.PlayerSecretEntry{
				PlayerID:      player.ID,
				CharacterName: player.Role.CharacterName,
				Secret:        player.Role.Secret,
				SpecialRole:   player.Role.SpecialRole,
			})
		}
	}

	// Semi-public reveals
	var semiPublicReveal []types.SemiPublicRevealEntry
	for _, sp := range world.Information.SemiPublic {
		semiPublicReveal = append(semiPublicReveal, types.SemiPublicRevealEntry{
			Info:          sp.Content,
			SharedBetween: sp.TargetPlayerIDs,
		})
	}

	// Undiscovered clues
	var undiscoveredClues []types.UndiscoveredClueEntry
	for _, clue := range world.Clues {
		cs, ok := state.ClueStates[clue.ID]
		if !ok || !cs.IsDiscovered {
			roomName := ""
			me := ece.gameState.GetMapEngine()
			if room := me.GetRoomByID(clue.RoomID); room != nil {
				roomName = room.Name
			}
			undiscoveredClues = append(undiscoveredClues, types.UndiscoveredClueEntry{
				Clue:     clue,
				RoomName: roomName,
			})
		}
	}

	// NPC secrets
	var npcSecrets []types.NPCSecretEntry
	for _, npc := range world.NPCs {
		if len(npc.HiddenInfo) > 0 {
			npcSecrets = append(npcSecrets, types.NPCSecretEntry{
				NPCName:    npc.Name,
				HiddenInfo: npc.HiddenInfo,
			})
		}
	}

	// Untriggered gimmicks
	var untriggeredGimmicks []types.GimmickReveal
	for _, gimmick := range world.Gimmicks {
		gs, ok := state.GimmickStates[gimmick.ID]
		if !ok || !gs.IsTriggered {
			untriggeredGimmicks = append(untriggeredGimmicks, types.GimmickReveal{
				GimmickID:   gimmick.ID,
				Name:        gimmick.Description,
				Description: gimmick.Effect,
				RoomID:      gimmick.RoomID,
				Condition:   gimmick.TriggerCondition,
			})
		}
	}

	return types.SecretReveal{
		PlayerSecrets:       playerSecrets,
		SemiPublicReveal:    semiPublicReveal,
		UndiscoveredClues:   undiscoveredClues,
		NPCSecrets:          npcSecrets,
		UntriggeredGimmicks: untriggeredGimmicks,
	}
}

// newBaseEvent creates a BaseEvent with visibility.
func newBaseEvent(scope string, roomID string, playerIDs []string) types.BaseEvent {
	return types.BaseEvent{
		ID:        generateID(),
		Timestamp: time.Now().UnixMilli(),
		Visibility: types.EventVisibility{
			Scope:     scope,
			RoomID:    roomID,
			PlayerIDs: playerIDs,
		},
	}
}

// generateID creates a simple unique ID based on timestamp.
func generateID() string {
	return time.Now().Format("20060102150405.000000000")
}

// Sentinel errors for end condition operations.
var (
	ErrNoActiveVote       = &EndConditionError{"no active vote"}
	ErrNoActiveConsensus  = &EndConditionError{"no active consensus"}
	ErrEndVoteAlreadyOpen = &EndConditionError{"end vote already open"}
	ErrNoEndProposal      = &EndConditionError{"no end proposal active"}
)

// EndConditionError represents an end condition engine error.
type EndConditionError struct {
	msg string
}

func (e *EndConditionError) Error() string { return e.msg }
