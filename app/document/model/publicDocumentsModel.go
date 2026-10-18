package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PublicDocumentsModel = (*customPublicDocumentsModel)(nil)

type (
	// PublicDocumentsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPublicDocumentsModel.
	PublicDocumentsModel interface {
		publicDocumentsModel
	}

	customPublicDocumentsModel struct {
		*defaultPublicDocumentsModel
	}
)

// NewPublicDocumentsModel returns a model for the database table.
func NewPublicDocumentsModel(conn sqlx.SqlConn, c cache.CacheConf) PublicDocumentsModel {
	return &customPublicDocumentsModel{
		defaultPublicDocumentsModel: newPublicDocumentsModel(conn, c),
	}
}
