package tcp

import (
	"github.com/grafana/sobek"
	"github.com/saniyar-dev/xk6-tcp/tcp/events"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// RootModule for TCPAPI extension
type RootModule struct{}

var _ modules.Module = &RootModule{}

// NewModuleInstance creates new module instance when called from k6 to return TCPAPI
func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &TCPAPI{
		vu: vu,
	}
}

// TCPAPI struct implements the api which is used on k6 extension
type TCPAPI struct {
	vu modules.VU
	// blobConstructor sobek.Value
}

var _ ExportedAPI = &TCPAPI{}

// Exports implements the modules.Instance interface's Exports
func (r *TCPAPI) Exports() modules.Exports {
	// r.blobConstructor = r.vu.Runtime().ToValue(r.blob)
	return modules.Exports{
		Named: map[string]interface{}{
			"TCP": r.init,
			// "Blob": r.blobConstructor,
		},
	}
}

func (r *TCPAPI) init(c sobek.ConstructorCall) *sobek.Object {
	t := &tcp{}
	rt := r.vu.Runtime()

	t = &tcp{
		vu: r.vu,

		url: t.url,
		obj: rt.NewObject(),

		doneCh:       make(chan struct{}),
		writeQueueCh: make(chan string),

		eventListeners: events.NewEventListeners(),
	}
	defineTCP(rt, t)

	// In this way the difference between having arguments in new TCP object and socket.open is doing it async or sync
	if len(c.Arguments) > 0 {
		err := t.parseURL(c.Argument(0))
		if err != nil {
			common.Throw(rt, err)
		}

		if err := t.open(t.url, tcpParams{}); err != nil {
			common.Throw(rt, err)
		}
	}

	return t.obj
}
