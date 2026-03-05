package ai

import (
	"context"
	"encoding/json"
	"fmt"

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
		MaxTokens:    8000,
	})
	if err != nil {
		return nil, fmt.Errorf("world generation AI call failed: %w", err)
	}

	var result schemas.WorldGeneration
	if err := json.Unmarshal(raw, &result); err != nil {
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
- At least %d rooms (player_count + 2), fully connected
- At least %d clues (player_count * 2)
- At least 1 semi-public information pair
- A timeout fallback end condition
- Brief, punchy text suitable for terminal display
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
			BriefingText:      gen.GameStructure.BriefingText,
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
