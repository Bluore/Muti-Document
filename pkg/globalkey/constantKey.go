package globalkey

const (
	DelStateNo  int64 = 0
	DelStateYes int64 = 1
)

// SessionStatus 会话运行状态 (int8 存储)
type SessionStatus int8

const (
	SessionStatusIdle        SessionStatus = iota // 空闲
	SessionStatusGenerating                       // 生成中
	SessionStatusDone                             // 已完成
	SessionStatusInterrupted                      // 已中断(等待人反馈)
	SessionStatusFailed                           // 失败
)
