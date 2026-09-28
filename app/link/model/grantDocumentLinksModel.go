package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GrantDocumentLinksModel = (*customGrantDocumentLinksModel)(nil)

type (
	// GrantDocumentLinksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGrantDocumentLinksModel.
	GrantDocumentLinksModel interface {
		grantDocumentLinksModel
	}

	customGrantDocumentLinksModel struct {
		*defaultGrantDocumentLinksModel
	}
)

// NewGrantDocumentLinksModel returns a model for the database table.
func NewGrantDocumentLinksModel(conn sqlx.SqlConn, c cache.CacheConf) GrantDocumentLinksModel {
	return &customGrantDocumentLinksModel{
		defaultGrantDocumentLinksModel: newGrantDocumentLinksModel(conn, c),
	}
}
