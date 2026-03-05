package session

import (
	"fmt"
	"math/rand"
)

var words = []string{
	"WOLF", "MOON", "STAR", "DARK", "FIRE", "IRON", "SILK", "JADE",
	"HAWK", "BEAR", "SAGE", "DUSK", "DAWN", "MIST", "VALE", "PEAK",
	"RUNE", "FANG", "VOID", "GALE", "THORN", "FROST", "SHADE", "STORM",
	"BLAZE", "CROWN", "DREAM", "EMBER", "FLAME", "GHOST", "HEART", "IVORY",
	"JEWEL", "KARMA", "LIGHT", "MAGIC", "NIGHT", "OCEAN", "PEARL", "QUEST",
	"ROYAL", "STONE", "TOWER", "UNITY", "VIGOR", "WATER", "XENON", "YOUTH",
	"AZURE", "BLADE", "CEDAR", "DELTA", "EAGLE", "FORGE", "GRAIN", "HAVEN",
	"INLET", "JOUST", "KNEEL", "LANCE", "MARSH", "NOBLE", "ONYX", "PRISM",
	"QUAKE", "RIDGE", "SABLE", "TITAN", "UMBRA", "VIPER", "WRATH", "ZEPHYR",
	"ARROW", "BIRCH", "CORAL", "DRUID", "ELFIN", "FJORD", "GLEAM", "HERON",
	"IVORY", "LYRIC", "MAPLE", "NEXUS", "ORBIT", "PLUME", "RAVEN", "SPIRE",
	"TROVE", "ULTRA", "VAULT", "WYVERN", "ABYSS", "BRIAR", "CREST", "DRYAD",
	"EPOCH", "FLORA", "GROVE", "HEATH",
}

// GenerateRoomCode returns a room code in the format "WORD-NNNN".
func GenerateRoomCode() string {
	word := words[rand.Intn(len(words))]
	number := rand.Intn(10000)
	return fmt.Sprintf("%s-%04d", word, number)
}
