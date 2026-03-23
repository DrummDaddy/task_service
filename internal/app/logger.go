package app

import "go.uber.org/zap"

func NewLogger(env string) *zap.Logger {
	if env == "production" {
		l, _ := zap.NewProduction()
		return l
	}
	l, _ := zap.NewDevelopment()
	return l
}
func ZapError(err error) zap.Field {
	return zap.Error(err)
}

func ZapString(key, val string) zap.Field {
	return zap.String(key, val)
}
func ZapInt(key string, val int) zap.Field {
	return zap.Int(key, val)
}
