package connection

import "sync"

// 机器人连接全局注册表: RPC 侧通过 BotId 获取对应的 BotConnection 并驱动其编辑。
var (
	botRegistryMu sync.RWMutex
	botRegistry   = make(map[string]*BotConnection, 100)
)

// RegisterBot 注册机器人连接
func RegisterBot(bot *BotConnection) {
	botRegistryMu.Lock()
	defer botRegistryMu.Unlock()
	botRegistry[bot.BotID] = bot
}

// GetBot 按 BotId 获取机器人连接
func GetBot(botID string) *BotConnection {
	botRegistryMu.RLock()
	defer botRegistryMu.RUnlock()
	return botRegistry[botID]
}

// UnregisterBot 移除机器人连接; 仅当注册表里仍是该连接时才删除, 避免误删重建的新连接
func UnregisterBot(bot *BotConnection) {
	botRegistryMu.Lock()
	defer botRegistryMu.Unlock()
	if current, ok := botRegistry[bot.BotID]; ok && current == bot {
		delete(botRegistry, bot.BotID)
	}
}
