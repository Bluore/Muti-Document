package grpcerr

var (
	codeMsg = make(map[ErrCode]string, 100)
)

func init() {
	// 全局错误码
	codeMsg[SERVER_COMMON_ERROR] = "rpc call failed"
	codeMsg[REQUEST_PARAM_ERROR] = "request params error"
	codeMsg[DB_ERROR] = "db error"
	codeMsg[REDIS_ERROR] = "redis error"

	// user模块
	codeMsg[USER_ID_EXIST_ERROR] = "user id already exist"
	codeMsg[USER_EMAIL_EXIST_ERROR] = "user email already exist"
	codeMsg[USER_NICKNAME_EXIST_ERROR] = "user nickname already exist"
	codeMsg[USER_NOT_EXIST_ERROR] = "user no exist"
	codeMsg[USER_TOKEN_GERENATE_ERROR] = "token generate error"
	codeMsg[USER_PASSWORD_ERROR] = "user password or login error"

	// document模块
	codeMsg[DOCUMENT_NOT_EXIST_ERROR] = "document no exist"
	codeMsg[DOCUMENT_NO_PERMISSION_ERROR] = "document permission denied"

	// link模块
	codeMsg[LINK_NOT_EXIST_ERROR] = "link no exist"
	codeMsg[LINK_EXPIRED_ERROR] = "link expired"
	codeMsg[LINK_NO_PERMISSION_ERROR] = "link permission denied"

	// agent模块
	codeMsg[AGENT_SESSION_NOT_EXIST_ERROR] = "agent session no exist"
	codeMsg[AGENT_SESSION_BUSY_ERROR] = "agent session is generating"
}
