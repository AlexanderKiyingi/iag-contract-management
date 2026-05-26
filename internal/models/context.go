package models

import "context"

type ctxKeySession struct{}

func WithRequestSession(ctx context.Context, sess Session) context.Context {
	return context.WithValue(ctx, ctxKeySession{}, sess)
}

func RequestSession(ctx context.Context) (Session, bool) {
	sess, ok := ctx.Value(ctxKeySession{}).(Session)
	return sess, ok
}
