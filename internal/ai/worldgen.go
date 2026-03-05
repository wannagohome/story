package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/story/internal/ai/provider"
	"github.com/anthropics/story/internal/shared/schemas"
	"github.com/anthropics/story/internal/shared/types"
)

// WorldGenerator handles AI-powered world generation.
type WorldGenerator struct {
	provider provider.AIProvider
}

// NewWorldGenerator creates a new WorldGenerator with the given AI provider.
func NewWorldGenerator(p provider.AIProvider) *WorldGenerator {
	return &WorldGenerator{provider: p}
}

// Generate creates a new game world using AI.
func (wg *WorldGenerator) Generate(
	ctx context.Context,
	settings types.GameSettings,
	playerCount int,
	themeKeyword string,
) (*types.World, error) {
	userPrompt := buildWorldGenUserPrompt(playerCount, themeKeyword, settings)

	raw, err := wg.provider.GenerateStructured(ctx, provider.StructuredRequest{
		SystemPrompt: worldGenSystemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.8,
		MaxTokens:    16000,
	})
	if err != nil {
		return nil, fmt.Errorf("world generation AI call failed: %w", err)
	}

	// Normalize the JSON to fix common AI output quirks before strict parsing.
	normalized, err := normalizeWorldGenJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("world generation normalize failed: %w", err)
	}

	var result schemas.WorldGeneration
	if err := json.Unmarshal(normalized, &result); err != nil {
		return nil, fmt.Errorf("world generation parse failed: %w", err)
	}

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("world generation validation failed: %w", err)
	}

	if len(result.Characters.PlayerRoles) != playerCount {
		return nil, fmt.Errorf("player role count (%d) does not match player count (%d)",
			len(result.Characters.PlayerRoles), playerCount)
	}

	return transformToWorld(&result), nil
}

// buildWorldGenUserPrompt constructs the user prompt for world generation.
func buildWorldGenUserPrompt(playerCount int, themeKeyword string, settings types.GameSettings) string {
	prompt := fmt.Sprintf("Create a game world for %d players.\n", playerCount)

	if themeKeyword != "" {
		prompt += fmt.Sprintf("Theme hint: %s. Use this as inspiration but don't be constrained by it.\n", themeKeyword)
	} else {
		prompt += "Genre, setting, and events are all up to you. Be creative.\n"
	}

	prompt += fmt.Sprintf("Time limit: %d minutes.\n", settings.TimeoutMinutes)

	if settings.HasGM {
		prompt += "This game has a GM (Game Master) who narrates events.\n"
	} else {
		prompt += "This game has no GM. The world must be self-driving.\n"
	}

	if settings.HasNPC {
		prompt += "Include NPCs that players can interact with.\n"
	} else {
		prompt += "No NPCs in this game. Player-to-player interaction only.\n"
	}

	prompt += fmt.Sprintf(`
Requirements:
- Exactly %d player roles with distinct personal goals and secrets
- At least %d rooms (player_count + 2), fully connected via bidirectional connections
- At least %d clues (player_count * 2)
- At least 1 semi-public information pair
- Brief, punchy text suitable for terminal display

Your output MUST be a single JSON object matching this exact structure:
{
  "meta": {"theme": "string", "setting": "string", "estimatedDuration": 20, "hasGM": true, "hasNPC": true},
  "world": {"title": "string (max 24 chars)", "synopsis": "string (2 sentences max)", "atmosphere": "string"},
  "gameStructure": {
    "concept": "string", "coreConflict": "string", "progressionStyle": "string",
    "commonGoal": "string or null", "briefingText": "string (5-7 lines)",
    "endConditions": [{"id": "string", "description": "string", "triggerType": "timeout|vote|consensus|event|ai_judgment", "triggerCriteria": {}, "isFallback": true}],
    "winConditions": [{"description": "string", "evaluationCriteria": "string"}],
    "requiredSystems": ["string"]
  },
  "map": {
    "rooms": [{"id": "room-xxx", "name": "string", "description": "string", "type": "public|private|secret", "items": [], "npcIds": ["npc-xxx"], "clueIds": ["clue-xxx"]}],
    "connections": [{"roomA": "room-xxx", "roomB": "room-yyy", "bidirectional": true}]
  },
  "characters": {
    "playerRoles": [{"id": "role-xxx", "characterName": "string", "background": "string", "secret": "string", "specialRole": "string or empty",
      "personalGoals": [{"id": "goal-xxx", "description": "string", "evaluationHint": "string", "entityRefs": []}],
      "relationships": [{"targetCharacterName": "string", "description": "string"}]}],
    "npcs": [{"id": "npc-xxx", "name": "string", "currentRoomId": "room-xxx", "persona": "string",
      "knownInfo": ["string"], "hiddenInfo": ["string"], "behaviorPrinciple": "string", "initialTrust": 0.5, "gimmick": null}]
  },
  "clues": [{"id": "clue-xxx", "name": "string", "description": "string", "roomId": "room-xxx", "discoverCondition": "string", "relatedClueIds": []}],
  "gimmicks": [],
  "information": {
    "public": {"title": "string", "synopsis": "string", "characterList": [{"name": "string", "publicDescription": "string"}],
      "relationships": "string", "mapOverview": "string", "npcList": [{"name": "string", "location": "string"}], "gameRules": "string"},
    "semiPublic": [{"id": "sp-xxx", "targetPlayerIds": ["role-xxx", "role-yyy"], "content": "string"}],
    "private": [{"playerId": "role-xxx", "additionalSecrets": ["string"]}]
  }
}

CRITICAL: estimatedDuration must be 10-30. At least one endCondition must have "isFallback": true. All roomIds, npcIds, clueIds must cross-reference correctly. semiPublic targetPlayerIds must use role IDs.
`, playerCount, playerCount+2, playerCount*2)

	return prompt
}

// transformToWorld converts a WorldGeneration schema to a World type.
func transformToWorld(gen *schemas.WorldGeneration) *types.World {
	world := &types.World{
		Title:      gen.World.Title,
		Synopsis:   gen.World.Synopsis,
		Atmosphere: gen.World.Atmosphere,
		GameStructure: types.GameStructure{
			Concept:           gen.GameStructure.Concept,
			CoreConflict:      gen.GameStructure.CoreConflict,
			ProgressionStyle:  gen.GameStructure.ProgressionStyle,
			CommonGoal:        gen.GameStructure.CommonGoal,
			EstimatedDuration: gen.Meta.EstimatedDuration,
			BriefingText:      string(gen.GameStructure.BriefingText),
		},
	}

	// End conditions
	for _, ec := range gen.GameStructure.EndConditions {
		world.GameStructure.EndConditions = append(world.GameStructure.EndConditions, types.EndCondition{
			ID:              ec.ID,
			Description:     ec.Description,
			TriggerType:     ec.TriggerType,
			TriggerCriteria: ec.TriggerCriteria,
			IsFallback:      ec.IsFallback,
		})
	}

	// Win conditions
	for _, wc := range gen.GameStructure.WinConditions {
		world.GameStructure.WinConditions = append(world.GameStructure.WinConditions, types.WinCondition{
			Description:        wc.Description,
			EvaluationCriteria: wc.EvaluationCriteria,
		})
	}

	// Required systems
	for _, rs := range gen.GameStructure.RequiredSystems {
		world.GameStructure.RequiredSystems = append(world.GameStructure.RequiredSystems, types.RequiredSystem(rs))
	}

	// Map
	world.Map = types.GameMap{}
	for _, r := range gen.Map.Rooms {
		room := types.Room{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Type:        r.Type,
			NPCIDs:      r.NPCIDs,
			ClueIDs:     r.ClueIDs,
		}
		for _, item := range r.Items {
			room.Items = append(room.Items, types.Item{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				OwnerID:     item.OwnerID,
				IsKey:       item.IsKey,
			})
		}
		world.Map.Rooms = append(world.Map.Rooms, room)
	}
	for _, c := range gen.Map.Connections {
		world.Map.Connections = append(world.Map.Connections, types.Connection{
			RoomA:         c.RoomA,
			RoomB:         c.RoomB,
			Bidirectional: c.Bidirectional,
		})
	}

	// Player roles
	for _, pr := range gen.Characters.PlayerRoles {
		role := types.PlayerRole{
			ID:            pr.ID,
			CharacterName: pr.CharacterName,
			Background:    pr.Background,
			Secret:        pr.Secret,
			SpecialRole:   pr.SpecialRole,
		}
		for _, pg := range pr.PersonalGoals {
			role.PersonalGoals = append(role.PersonalGoals, types.PersonalGoal{
				ID:             pg.ID,
				Description:    pg.Description,
				EvaluationHint: pg.EvaluationHint,
				EntityRefs:     pg.EntityRefs,
			})
		}
		for _, rel := range pr.Relationships {
			role.Relationships = append(role.Relationships, types.Relationship{
				TargetCharacterName: rel.TargetCharacterName,
				Description:         rel.Description,
			})
		}
		world.PlayerRoles = append(world.PlayerRoles, role)
	}

	// NPCs
	for _, n := range gen.Characters.NPCs {
		npc := types.NPC{
			ID:                n.ID,
			Name:              n.Name,
			CurrentRoomID:     n.CurrentRoomID,
			Persona:           n.Persona,
			KnownInfo:         n.KnownInfo,
			HiddenInfo:        n.HiddenInfo,
			BehaviorPrinciple: n.BehaviorPrinciple,
			InitialTrust:      n.InitialTrust,
		}
		if n.Gimmick != nil {
			npc.Gimmick = &types.NPCGimmick{
				Description:      n.Gimmick.Description,
				TriggerCondition: n.Gimmick.TriggerCondition,
				Effect:           n.Gimmick.Effect,
			}
		}
		if npc.InitialTrust == 0 {
			npc.InitialTrust = 0.5
		}
		world.NPCs = append(world.NPCs, npc)
	}

	// Clues
	for _, c := range gen.Clues {
		world.Clues = append(world.Clues, types.Clue{
			ID:                c.ID,
			Name:              c.Name,
			Description:       c.Description,
			RoomID:            c.RoomID,
			DiscoverCondition: c.DiscoverCondition,
			RelatedClueIDs:    c.RelatedClueIDs,
		})
	}

	// Gimmicks
	for _, g := range gen.Gimmicks {
		world.Gimmicks = append(world.Gimmicks, types.Gimmick{
			ID:               g.ID,
			Description:      g.Description,
			RoomID:           g.RoomID,
			TriggerCondition: g.TriggerCondition,
			Effect:           g.Effect,
		})
	}

	// Information layers
	world.Information = types.InformationLayers{
		Public: types.PublicInfo{
			Title:         gen.Information.Public.Title,
			Synopsis:      gen.Information.Public.Synopsis,
			Relationships: gen.Information.Public.Relationships,
			MapOverview:   gen.Information.Public.MapOverview,
			GameRules:     gen.Information.Public.GameRules,
		},
	}
	for _, cl := range gen.Information.Public.CharacterList {
		world.Information.Public.CharacterList = append(world.Information.Public.CharacterList, types.CharacterListEntry{
			Name:              cl.Name,
			PublicDescription: cl.PublicDescription,
		})
	}
	for _, nl := range gen.Information.Public.NPCList {
		world.Information.Public.NPCList = append(world.Information.Public.NPCList, types.NPCListEntry{
			Name:     nl.Name,
			Location: nl.Location,
		})
	}
	for _, sp := range gen.Information.SemiPublic {
		world.Information.SemiPublic = append(world.Information.SemiPublic, types.SemiPublicInfo{
			ID:              sp.ID,
			TargetPlayerIDs: sp.TargetPlayerIDs,
			Content:         sp.Content,
		})
	}
	for _, p := range gen.Information.Private {
		world.Information.Private = append(world.Information.Private, types.PrivateInfo{
			PlayerID:          p.PlayerID,
			AdditionalSecrets: p.AdditionalSecrets,
		})
	}

	return world
}

// normalizeWorldGenJSON fixes common AI output quirks where fields have wrong
// types (e.g., string instead of array, array instead of string).
func normalizeWorldGenJSON(raw json.RawMessage) (json.RawMessage, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return raw, nil // return as-is, let strict parser handle it
	}

	// Fix information.public fields
	if info, ok := data["information"].(map[string]interface{}); ok {
		if pub, ok := info["public"].(map[string]interface{}); ok {
			// characterList: ensure it's an array of objects
			if cl, ok := pub["characterList"]; ok {
				pub["characterList"] = ensureObjectArray(cl, []string{"name", "publicDescription"})
			}
			// npcList: ensure it's an array of objects
			if nl, ok := pub["npcList"]; ok {
				pub["npcList"] = ensureObjectArray(nl, []string{"name", "location"})
			}
			// Ensure string fields are strings (not arrays)
			for _, field := range []string{"title", "synopsis", "relationships", "mapOverview", "gameRules"} {
				pub[field] = ensureString(pub[field])
			}
		}
		// Ensure semiPublic exists as array
		if _, ok := info["semiPublic"]; !ok {
			info["semiPublic"] = []interface{}{}
		}
		// Ensure private exists as array
		if _, ok := info["private"]; !ok {
			info["private"] = []interface{}{}
		}
	}

	// Fix world fields
	if w, ok := data["world"].(map[string]interface{}); ok {
		for _, field := range []string{"title", "synopsis", "atmosphere"} {
			w[field] = ensureString(w[field])
		}
	}

	// Fix gameStructure fields
	if gs, ok := data["gameStructure"].(map[string]interface{}); ok {
		for _, field := range []string{"concept", "coreConflict", "progressionStyle"} {
			gs[field] = ensureString(gs[field])
		}
	}

	// Fix meta.estimatedDuration: ensure it's at least 10
	if meta, ok := data["meta"].(map[string]interface{}); ok {
		if dur, ok := meta["estimatedDuration"].(float64); ok && dur < 10 {
			meta["estimatedDuration"] = 15.0
		}
		if _, ok := meta["estimatedDuration"]; !ok {
			meta["estimatedDuration"] = 20.0
		}
	}

	// Fix gameStructure.endConditions: ensure at least one with isFallback
	if gs, ok := data["gameStructure"].(map[string]interface{}); ok {
		endConds, _ := gs["endConditions"].([]interface{})
		if len(endConds) == 0 {
			gs["endConditions"] = []interface{}{
				map[string]interface{}{
					"id":              "end-timeout",
					"description":     "Game times out",
					"triggerType":     "timeout",
					"triggerCriteria": map[string]interface{}{},
					"isFallback":      true,
				},
			}
		} else {
			// Ensure at least one has isFallback
			hasFallback := false
			for _, ec := range endConds {
				if ecMap, ok := ec.(map[string]interface{}); ok {
					if fb, ok := ecMap["isFallback"].(bool); ok && fb {
						hasFallback = true
						break
					}
				}
			}
			if !hasFallback {
				// Mark the last one as fallback
				if lastEC, ok := endConds[len(endConds)-1].(map[string]interface{}); ok {
					lastEC["isFallback"] = true
				}
			}
		}
	}

	return json.Marshal(data)
}

// ensureString coerces a value to a string. Arrays of strings are joined with newlines.
func ensureString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		var parts []string
		for _, elem := range val {
			if s, ok := elem.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, "\n")
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

// ensureObjectArray ensures a value is an array of objects with the given fields.
// Handles: string → [{field[0]: string}], ["s1","s2"] → [{field[0]: "s1"}, ...],
// [obj1, obj2] → as-is.
func ensureObjectArray(v interface{}, fields []string) interface{} {
	makeObj := func(name string) map[string]interface{} {
		obj := map[string]interface{}{fields[0]: name}
		for _, f := range fields[1:] {
			obj[f] = ""
		}
		return obj
	}

	switch val := v.(type) {
	case []interface{}:
		// Check if elements are strings and convert to objects
		result := make([]interface{}, 0, len(val))
		for _, elem := range val {
			switch e := elem.(type) {
			case string:
				if len(fields) > 0 {
					result = append(result, makeObj(e))
				}
			case map[string]interface{}:
				result = append(result, e)
			default:
				result = append(result, elem)
			}
		}
		return result
	case string:
		if len(fields) > 0 {
			return []interface{}{makeObj(val)}
		}
		return []interface{}{}
	default:
		return []interface{}{}
	}
}
