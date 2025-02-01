package tcp

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/grafana/sobek"
	"github.com/mstoykov/k6-taskqueue-lib/taskqueue"
	"github.com/saniyar-dev/xk6-tcp/tcp/events"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/metrics"
)

// Thing interface is an interface which all the thing needs to implement this.
type Thing interface {
	parseURL(sobek.Value) error
	addEventListener(string, func(sobek.Value) (sobek.Value, error))
}

type tcp struct {
	vu modules.VU

	url            *url.URL
	conn           *net.Conn
	tagsAndMeta    *metrics.TagsAndMeta
	tq             *taskqueue.TaskQueue
	builtinMetrics *metrics.BuiltinMetrics
	obj            *sobek.Object
	started        time.Time

	doneCh       chan struct{}
	writeQueueCh chan string

	eventListeners *events.EventListeners
}

var _ Thing = &tcp{}

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
		"on", rt.ToValue(t.addEventListener), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, t.obj.DefineDataProperty(
		"write", rt.ToValue(t.writeAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, t.obj.DefineDataProperty(
		"open", rt.ToValue(t.openAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, t.obj.DefineDataProperty(
		"done", rt.ToValue(t.doneAsync), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))

	setOn := func(property string, el *events.EventListener) {
		if el == nil {
			// this is generally should not happen, but we're being defensive
			common.Throw(rt, fmt.Errorf("not supported on-handler '%s'", property))
		}

		must(rt, t.obj.DefineAccessorProperty(
			property, rt.ToValue(func() sobek.Value {
				return rt.ToValue(el.GetOn)
			}), rt.ToValue(func(call sobek.FunctionCall) sobek.Value {
				arg := call.Argument(0)

				// it's possible to unset handlers by setting them to null
				if arg == nil || sobek.IsUndefined(arg) || sobek.IsNull(arg) {
					el.SetOn(nil)

					return nil
				}

				fn, isFunc := sobek.AssertFunction(arg)
				if !isFunc {
					common.Throw(rt, fmt.Errorf("a value for '%s' should be callable", property))
				}

				el.SetOn(func(v sobek.Value) (sobek.Value, error) { return fn(sobek.Undefined(), v) })

				return nil
			}), sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	}

	setOn("onopen", t.eventListeners.GetType(events.OPEN))
	setOn("ondata", t.eventListeners.GetType(events.DATA))
	setOn("onclose", t.eventListeners.GetType(events.CLOSE))
	setOn("onerror", t.eventListeners.GetType(events.ERROR))
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

	if err := t.eventListeners.Add(event, handler); err != nil {
		t.vu.State().Logger.Warnf("can't add event handler: %s", err)
	}
}
