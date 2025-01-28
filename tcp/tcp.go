package tcp

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

type ClientModule interface {
	define(*sobek.Runtime)
}

type RootModule struct{}

var _ modules.Module = &RootModule{}

func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &TCPAPI{
		vu: vu,
	}
}

type TCPAPI struct {
	vu modules.VU
	// blobConstructor sobek.Value
}

var _ modules.Instance = &TCPAPI{}

// Exports implements the modules.Instance interface's Exports
func (r *TCPAPI) Exports() modules.Exports {
	// r.blobConstructor = r.vu.Runtime().ToValue(r.blob)
	return modules.Exports{
		Named: map[string]interface{}{
			"TCP": r.tcp,
			// "Blob": r.blobConstructor,
		},
	}
}

type tcp struct {
	vu modules.VU

	url *url.URL
	// conn *net.Conn
	// tagsAndMeta    *metrics.TagsAndMeta
	// tq             *taskqueue.TaskQueue
	// builtinMetrics *metrics.BuiltinMetrics
	obj *sobek.Object
	// started time.Time

	done         chan struct{}
	writeQueueCh chan string

	eventListeners *eventListeners
}

func (r *TCPAPI) tcp(c sobek.ConstructorCall) *sobek.Object {
	rt := r.vu.Runtime()

	url, err := parseURL(c.Argument(0))
	if err != nil {
		common.Throw(rt, err)
	}

	t := &tcp{
		vu: r.vu,

		url: url,
		obj: rt.NewObject(),

		done:         make(chan struct{}),
		writeQueueCh: make(chan string),

		eventListeners: newEventListeners(),
	}
	defineTCP(rt, t)

	return t.obj
}

// parseURL parses and validate the url from the first constructor calls argument or returns an error
func parseURL(urlValue sobek.Value) (*url.URL, error) {
	if urlValue == nil || sobek.IsUndefined(urlValue) {
		return nil, errors.New("TCP requires a url")
	}

	urlString := urlValue.String()
	url, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("TCP requires valid url, but got %q which resulted in %w", urlString, err)
	}

	return url, nil
}

// defineTCP is defining tcp object in javascrip runtime using sobek engine, so that if you write:
//
//	const socket = new TCP(url, params)
//	socket.on('data', data => {
//	  console.log(`received ${data}`);
//	  socket.close();
//	});
//
// engine knows what to call in golang program.
func defineTCP(rt *sobek.Runtime, t *tcp) {
	// TODO add more definition
	must(rt, t.obj.DefineDataProperty(
		"addEventListener", rt.ToValue(t.addEventListener), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, t.obj.DefineDataProperty(
		"write", rt.ToValue(t.writeAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
}

func (t *tcp) write(m string) error {
	fmt.Printf("The message is %s", m)
	return nil
}

func (t *tcp) writeAsync(m string) *sobek.Promise {
	enqCallback := t.vu.RegisterCallback()
	p, resolve, reject := t.vu.Runtime().NewPromise()

	go func() {
		err := t.write(m)
		enqCallback(func() error {
			if err != nil {
				if er := reject(err); er != nil {
					return er
				}
			}
			if er := resolve("success"); er != nil {
				return er
			}
			return nil
		})
	}()

	return p
}

// addEventListener adds event listeners on your tcp struct
func (t *tcp) addEventListener(event string, handler func(sobek.Value) (sobek.Value, error)) {
	// TODO support options https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener#parameters

	if handler == nil {
		common.Throw(t.vu.Runtime(), fmt.Errorf("handler for event type %q isn't a callable function", event))
	}

	if err := t.eventListeners.add(event, handler); err != nil {
		t.vu.State().Logger.Warnf("can't add event handler: %s", err)
	}
}
