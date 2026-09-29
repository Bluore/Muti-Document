package globalkey

const (
	// 公共模块

	// 用户模块
	RedisKeyUserVerifyKey   = "key:user:email:verify:code:%s"
	RedisKeyUserVerifyLimit = "key:user:email:verify:limit:%s"

	// 短链模块
	RedisKeyShortLinkId = "key:link:short_link:id"

	// 文档模块
	// RedisKeyDocumentChange 文档尚未落库的变更, 格式为文档id
	RedisKeyDocumentChange = "key:document:change:%s"
)
