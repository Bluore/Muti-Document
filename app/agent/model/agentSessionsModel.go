package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AgentSessionsModel = (*customAgentSessionsModel)(nil)

type (
	// AgentSessionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAgentSessionsModel.
	AgentSessionsModel interface {
		agentSessionsModel
	}

	customAgentSessionsModel struct {
		*defaultAgentSessionsModel
	}
)

// NewAgentSessionsModel returns a model for the database table.
func NewAgentSessionsModel(conn sqlx.SqlConn, c cache.CacheConf) AgentSessionsModel {
	return &customAgentSessionsModel{
		defaultAgentSessionsModel: newAgentSessionsModel(conn, c),
	}
}
