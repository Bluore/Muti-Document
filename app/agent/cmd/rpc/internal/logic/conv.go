package logic

import (
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/session"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/pb"
)

// toPbSession 会话元信息转 pb
func toPbSession(client *session.SessionClient) *pb.Session {
	meta := client.GetSession()
	return &pb.Session{
		SessionId:    meta.SessionID,
		SessionTitle: meta.SessionTitle,
		UserId:       meta.UserID,
	}
}
