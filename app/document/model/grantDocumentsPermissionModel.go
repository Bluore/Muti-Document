package model

import (
	"context"
	"fmt"

	"github.com/Bluore/mutidoc/pkg/globalkey"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GrantDocumentsPermissionModel = (*customGrantDocumentsPermissionModel)(nil)

type (
	// GrantDocumentsPermissionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGrantDocumentsPermissionModel.
	GrantDocumentsPermissionModel interface {
		grantDocumentsPermissionModel

		// FindOneByUserIdAndDocumentId 查询用户对指定文档的授权记录
		FindOneByUserIdAndDocumentId(ctx context.Context, userId, documentId string) (*GrantDocumentsPermission, error)
	}

	customGrantDocumentsPermissionModel struct {
		*defaultGrantDocumentsPermissionModel
	}
)

// NewGrantDocumentsPermissionModel returns a model for the database table.
func NewGrantDocumentsPermissionModel(conn sqlx.SqlConn, c cache.CacheConf) GrantDocumentsPermissionModel {
	return &customGrantDocumentsPermissionModel{
		defaultGrantDocumentsPermissionModel: newGrantDocumentsPermissionModel(conn, c),
	}
}

func (m *customGrantDocumentsPermissionModel) FindOneByUserIdAndDocumentId(ctx context.Context, userId, documentId string) (*GrantDocumentsPermission, error) {
	var resp GrantDocumentsPermission
	query := fmt.Sprintf("select %s from %s where `user_id` = ? and `document_id` = ? and del_state = ? limit 1", grantDocumentsPermissionRows, m.table)
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, userId, documentId, globalkey.DelStateNo)
	switch err {
	case nil:
		return &resp, nil
	case ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}
