package requestlogger

import (
	"flamingo/core/dingo"
	"flamingo/framework/event"
)

type Module struct{}

func (m *Module) Configure(injector *dingo.Injector) {
	injector.BindMulti((*event.Subscriber)(nil)).To(Logger{})
}
