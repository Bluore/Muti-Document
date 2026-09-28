package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ShortLinksModel = (*customShortLinksModel)(nil)

type (
	// ShortLinksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customShortLinksModel.
	ShortLinksModel interface {
		shortLinksModel
	}

	customShortLinksModel struct {
		*defaultShortLinksModel
	}
)

// NewShortLinksModel returns a model for the database table.
func NewShortLinksModel(conn sqlx.SqlConn, c cache.CacheConf) ShortLinksModel {
	return &customShortLinksModel{
		defaultShortLinksModel: newShortLinksModel(conn, c),
	}
}
