package entity

// BotIDBase is the starting peer ID for bot players.
// Real clients never reach this range.
const BotIDBase uint16 = 50000

// IsBotID returns true if the peer ID belongs to a bot player.
func IsBotID(id uint16) bool {
	return id >= BotIDBase
}
