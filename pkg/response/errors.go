package response

var (
	// 公共错误
	// 3位服务编号，4位业务错误编号
	ErrValidation = NewError(200, 1004001, "字段验证错误")
	ErrServer     = NewError(500, 1005001, "服务器内部出错")
)
