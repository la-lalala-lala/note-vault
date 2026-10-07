package utils

import "context"

type auditActorKey struct{}

type AuditActor struct {
	Account string
	IP      string
}

// WithAuditActor 将认证后的账号及可信来源 IP 传给业务层，不接收客户端传入的账号。
func WithAuditActor(ctx context.Context, account, ip string) context.Context {
	return context.WithValue(ctx, auditActorKey{}, AuditActor{Account: account, IP: ip})
}

func CurrentAuditActor(ctx context.Context) (AuditActor, bool) {
	actor, ok := ctx.Value(auditActorKey{}).(AuditActor)
	return actor, ok && actor.Account != ""
}
