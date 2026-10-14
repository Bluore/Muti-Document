package toolcallresp

import (
	"context"
	"errors"
	"fmt"
)

// 工具调用可能出现的错误, 供工具判定失败原因
var (
	// ErrDocumentIDRequired 缺少 document_id 参数
	ErrDocumentIDRequired = errors.New("document_id is required")
	// ErrUserNotIdentified 无法从上下文识别当前用户
	ErrUserNotIdentified = errors.New("user not identified")
	// ErrDocumentNoPermission 当前用户对文档没有权限
	ErrDocumentNoPermission = errors.New("no permission for document")
	// ErrDocumentNotExist 文档不存在
	ErrDocumentNotExist = errors.New("document not exist")
	// ErrDocumentQueryFailed 查询文档失败
	ErrDocumentQueryFailed = errors.New("query document failed")
	// ErrDocumentListFailed 查询文档列表失败
	ErrDocumentListFailed = errors.New("list document failed")
)

// errorTexts 已知错误到自然语言说明的映射, 按注册顺序匹配
var errorTexts = []struct {
	err error
	msg string
}{
	{ErrDocumentIDRequired, "请提供要查询的文档 ID。"},
	{ErrUserNotIdentified, "暂时无法识别当前用户，请重新登录后再试。"},
	{ErrDocumentNoPermission, "你没有权限查看这篇文档，无法返回它的内容。"},
	{ErrDocumentNotExist, "没有找到对应的文档，请确认文档 ID 是否正确。"},
	{ErrDocumentQueryFailed, "查询文档时出了点问题，请稍后再试。"},
	{ErrDocumentListFailed, "获取文档列表时出了点问题，请稍后再试。"},
}

// NewDocumentIDRequiredErr 缺少 document_id 参数
func NewDocumentIDRequiredErr() (string, error) {
	return NewError(ErrDocumentIDRequired)
}

// NewUserNotIdentifiedErr 无法识别当前用户
func NewUserNotIdentifiedErr() (string, error) {
	return NewError(ErrUserNotIdentified)
}

// NewDocumentNoPermissionErr 当前用户对文档没有权限
func NewDocumentNoPermissionErr() (string, error) {
	return NewError(ErrDocumentNoPermission)
}

// NewDocumentNotExistErr 文档不存在
func NewDocumentNotExistErr() (string, error) {
	return NewError(ErrDocumentNotExist)
}

// NewDocumentQueryFailedErr 查询文档失败, cause 为底层原因
func NewDocumentQueryFailedErr(cause error) (string, error) {
	if cause == nil {
		return NewError(ErrDocumentQueryFailed)
	}
	return NewError(fmt.Errorf("%w: %v", ErrDocumentQueryFailed, cause))
}

// NewDocumentListFailedErr 查询文档列表失败, cause 为底层原因
func NewDocumentListFailedErr(cause error) (string, error) {
	if cause == nil {
		return NewError(ErrDocumentListFailed)
	}
	return NewError(fmt.Errorf("%w: %v", ErrDocumentListFailed, cause))
}

// NewError 把工具调用错误转成面向模型的自然语言说明, 并原样返回错误
func NewError(err error) (string, error) {
	return Naturalize(err), err
}

// Naturalize 把错误翻译成自然语言说明, 未知错误回退为通用文案
func Naturalize(err error) string {
	if err == nil {
		return ""
	}
	for _, e := range errorTexts {
		if errors.Is(err, e.err) {
			return e.msg
		}
	}
	return fmt.Sprintf("工具调用失败：%v", err)
}

// ErrorHandler 供 eino 工具错误处理中间件使用, 把错误转成自然语言结果
func ErrorHandler(_ context.Context, err error) string {
	return Naturalize(err)
}
