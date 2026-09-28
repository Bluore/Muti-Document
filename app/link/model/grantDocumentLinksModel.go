package model

import (
	"context"
	"fmt"

	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GrantDocumentLinksModel = (*customGrantDocumentLinksModel)(nil)

type (
	// GrantDocumentLinksModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGrantDocumentLinksModel.
	GrantDocumentLinksModel interface {
		grantDocumentLinksModel

		// FindOneByDocumentIdAndUserId 查询指定文档与用户的授权链接
		FindOneByDocumentIdAndUserId(ctx context.Context, documentId, userId string) (*GrantDocumentLinks, error)
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

func (m *customGrantDocumentLinksModel) FindOneByDocumentIdAndUserId(ctx context.Context, documentId, userId string) (*GrantDocumentLinks, error) {
	var resp GrantDocumentLinks
	query := fmt.Sprintf("select %s from %s where `document_id` = ? and `user_id` = ? and del_state = ? limit 1", grantDocumentLinksRows, m.table)
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, documentId, userId, globalkey.DelStateNo)
	switch err {
	case nil:
		return &resp, nil
	case ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
