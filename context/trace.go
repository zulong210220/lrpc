package context

var (
	keyTraceId = "metaTraceId"
)

func SetTraceId(ctx *Context, traceId string) {
	ctx.SetValue(keyTraceId, traceId)
}

func GetTraceId(ctx *Context) string {
	return ctx.Value(keyTraceId).(string)
}
