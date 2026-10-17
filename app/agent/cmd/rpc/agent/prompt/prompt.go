package prompt

import _ "embed"

// MainAgent 入口智能体系统提示词, 定义身份、业务边界、保密要求与工作方式
//
//go:embed mainAgent.md
var MainAgent string

// SearchAgent 文档检索智能体系统提示词, 负责复杂的多步文档任务
//
//go:embed searchAgent.md
var SearchAgent string

// EditAgent 文档编辑智能体系统提示词, 负责按任务改写指定文档
//
//go:embed editAgent.md
var EditAgent string
