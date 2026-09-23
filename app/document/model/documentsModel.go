package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ DocumentsModel = (*customDocumentsModel)(nil)

type (
	// DocumentsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDocumentsModel.
	DocumentsModel interface {
		documentsModel
	}

	customDocumentsModel struct {
		*defaultDocumentsModel
	}
)

// NewDocumentsModel returns a model for the database table.
func NewDocumentsModel(conn sqlx.SqlConn, c cache.CacheConf) DocumentsModel {
	return &customDocumentsModel{
		defaultDocumentsModel: newDocumentsModel(conn, c),
	}
}
