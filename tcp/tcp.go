package tcp

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/grafana/sobek"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// ExportedAPI interface is an interface which all exported api needs to implement this interface
type ExportedAPI interface {
	init(sobek.ConstructorCall) *sobek.Object
}

// Thing interface is an interface which all the thing needs to implement this.
type Thing interface {
	parseURL(sobek.Value) error
}

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

var (
	_ modules.Instance = &TCPAPI{}
	_ ExportedAPI      = &TCPAPI{}
)

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

type tcp struct {
	vu modules.VU

	url *url.URL
	// conn *net.Conn
	// tagsAndMeta    *metrics.TagsAndMeta
	// tq             *taskqueue.TaskQueue
	// builtinMetrics *metrics.BuiltinMetrics
	obj *sobek.Object
	// started time.Time

	doneCh       chan struct{}
	writeQueueCh chan string

	eventListeners *eventListeners
}

var _ Thing = &tcp{}

func (r *TCPAPI) init(c sobek.ConstructorCall) *sobek.Object {
	t := &tcp{}
	rt := r.vu.Runtime()

	// TODO you can mutate url in this function and there is no need to return the value
	err := t.parseURL(c.Argument(0))
	if err != nil {
		common.Throw(rt, err)
	}

	t = &tcp{
		vu: r.vu,

		url: t.url,
		obj: rt.NewObject(),

		doneCh:       make(chan struct{}),
		writeQueueCh: make(chan string),

		eventListeners: newEventListeners(),
	}
	defineTCP(rt, t)

	return t.obj
}

// parseURL parses and validate the url from the first constructor calls argument or returns an error
func (t *tcp) parseURL(urlValue sobek.Value) error {
	if urlValue == nil || sobek.IsUndefined(urlValue) {
		return errors.New("TCP requires a url")
	}

	urlString := urlValue.String()
	url, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("TCP requires valid url, but got %q which resulted in %w", urlString, err)
	}

	t.url = url
	return nil
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
	must(rt, t.obj.DefineDataProperty(
		"open", rt.ToValue(t.openAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, t.obj.DefineDataProperty(
		"done", rt.ToValue(t.doneAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
}

func (t *tcp) done() error {
	// TODO write done function
	// you should actually close the socket, but should you erase the all other properties?? do we need them after this?
	// memory and garbage collecter issues should be handled here
	fmt.Printf("close the socket")
	return nil
}

func (t *tcp) doneAsync() *sobek.Promise {
	enqCallback := t.vu.RegisterCallback()
	p, resolve, reject := t.vu.Runtime().NewPromise()

	go func() {
		err := t.done()
		enqCallback(func() error {
			if err != nil {
				if er := reject(err); er != nil {
					return er
				}
			}
			if er := resolve("success opening socket."); er != nil {
				return er
			}
			return nil
		})
	}()

	return p
}

func (t *tcp) open(url url.URL, params tcpParams) error {
	// TODO write open function
	// mayby we can now have the socket net.Conn on t struct?
	// it's exactly like init function from TCPAPI
	fmt.Printf("open tcp socket with url: %s and params: %s", url.String(), params)
	return nil
}

func (t *tcp) openAsync(url url.URL, params tcpParams) *sobek.Promise {
	enqCallback := t.vu.RegisterCallback()
	p, resolve, reject := t.vu.Runtime().NewPromise()

	go func() {
		err := t.open(url, params)
		enqCallback(func() error {
			if err != nil {
				if er := reject(err); er != nil {
					return er
				}
			}
			if er := resolve("success opening socket."); er != nil {
				return er
			}
			return nil
		})
	}()

	return p
}

func (t *tcp) write(m string) error {
	// TODO write write function
	// mayby we can now have the socket net.Conn on t struct and use it here??
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
			if er := resolve("success writing on socket"); er != nil {
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
