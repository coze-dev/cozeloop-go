package consts

type SpanFinishEvent string

const (
	SpanFinishEventSpanQueueEntryRate SpanFinishEvent = "queue_manager.span_entry.rate"
	SpanFinishEventFileQueueEntryRate SpanFinishEvent = "queue_manager.file_entry.rate"

	SpanFinishEventFlushSpanRate SpanFinishEvent = "exporter.span_flush.rate"
	SpanFinishEventFlushFileRate SpanFinishEvent = "exporter.file_flush.rate"
)

type FinishEventInfo struct {
	EventType   SpanFinishEvent
	IsEventFail bool
	ItemNum     int // maybe multiple span is processed in one event
	DetailMsg   string
	ExtraParams *FinishEventInfoExtra
}

type FinishEventInfoExtra struct {
	IsRootSpan bool
	LatencyMs  int64
}
