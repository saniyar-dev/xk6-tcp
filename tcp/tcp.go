package tcp

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/grafana/sobek"
	"github.com/mstoykov/k6-taskqueue-lib/taskqueue"
	"github.com/saniyar-dev/xk6-tcp/tcp/events"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

// Thing interface is an interface which all the thing needs to implement this.
type Thing interface {
	parseURL(sobek.Value) error
	addEventListener(string, func(sobek.Value) (sobek.Value, error))
	newEvent(string, time.Time) *sobek.Object
}

// ReadyState is tcp socket specification's readystate
type ReadyState uint8

const (
	// CONNECTING is the state while the tcp socket is connecting
	CONNECTING ReadyState = iota
	// OPEN is the state after the tcp socket is established and before it starts closing
	OPEN
	// CLOSING is while the tcp socket is closing but is *not* closed yet
	CLOSING
	// CLOSED is when the tcp socket is finally closed
	CLOSED
)

type tcp struct {
	vu modules.VU

	url  string
	conn net.Conn
	// tagsAndMeta    *metrics.TagsAndMeta
	tq *taskqueue.TaskQueue
	// builtinMetrics *metrics.BuiltinMetrics
	obj *sobek.Object
	// started time.Time

	doneCh       chan struct{}
	writeQueueCh chan string

	eventListeners *events.EventListeners

	readyState ReadyState
}

var _ Thing = &tcp{}

// parseURL parses and validate the url from the first constructor calls argument or returns an error
func (t *tcp) parseURL(urlValue sobek.Value) error {
	addr := urlValue.String()
	u, err := url.Parse(addr)
	if err == nil && u.Host != "" {
		// Use the host from URL (includes port if specified)
		addr = u.Host
	}

	// Split into host:port components
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("TCP requires valid url, but got %q which resulted in %w", addr, err)
	}
	t.url = net.JoinHostPort(host, port)
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

type message struct {
	data      []byte
	timestamp time.Time
}

func (t *tcp) queueMessage(message *message) {
	t.tq.Queue(func() error {
		if t.readyState != OPEN {
			return nil
		}

		rt := t.vu.Runtime()
		ev := t.newEvent(events.DATA, message.timestamp)
		must(
			rt,
			ev.DefineDataProperty("data", rt.ToValue(string(message.data)), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE),
		)
		must(
			rt,
			ev.DefineDataProperty("origin", rt.ToValue(t.url), sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE),
		)

		for _, dataListener := range t.eventListeners.All(events.DATA) {
			if _, err := dataListener(ev); err != nil {
				_ = t.conn.Close()                   // TODO log it?
				_ = t.connectionClosedWithError(err) // TODO log it?
				return err
			}
		}
		return nil
	})
}

func (t *tcp) loop() {
	for {
		b := make([]byte, 1024)
		_, err := t.conn.Read(b)
		if err == nil {
			t.queueMessage(&message{
				data:      b,
				timestamp: time.Now(),
			})
			continue
		}

		t.tq.Queue(func() error {
			_ = t.conn.Close()
			_ = t.connectionClosedWithError(err)
			return nil
		})
	}
}

func (t *tcp) connectionConnected() error {
	if t.readyState != CONNECTING {
		return nil
	}
	t.readyState = OPEN
	return t.callOpenListeneres(time.Now())
}

func (t *tcp) connectionClosedWithError(err error) error {
	if t.readyState == CLOSED {
		return nil
	}
	t.readyState = CLOSED
	// close(t.done)

	if err != nil {
		if errList := t.callErrorListeners(err, time.Now()); errList != nil {
			return errList // TODO ... still call the close listeners ?!?
		}
	}
	return t.callEventListeners(events.CLOSE, time.Now())
}

func (t *tcp) done() error {
	// TODO write done function
	// you should actually close the socket, but should you erase the all other properties?? do we need them after this?
	// memory and garbage collecter issues should be handled here

	if err := t.conn.Close(); err != nil {
		return err
	}
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

func (t *tcp) open(url string, params tcpParams) error {
	// TODO write open function
	// mayby we can now have the socket net.Conn on t struct?
	// it's exactly like init function from TCPAPI
	ctx := t.vu.Context()
	td := &net.Dialer{}

	conn, connErr := td.DialContext(ctx, "tcp", url)
	if connErr != nil {
		return connErr
	}
	t.conn = conn

	if connErr != nil {
		return connErr
	}

	go t.loop()
	t.tq.Queue(func() error {
		return t.connectionConnected()
	})

	fmt.Printf("open tcp socket with url: %s and params: %s", url, params)
	return nil
}

func (t *tcp) openAsync(url string, params tcpParams) *sobek.Promise {
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
	b, err := common.ToBytes(m)
	if err != nil {
		return err
	}
	_, err = t.conn.Write(b)
	if err != nil {
		return err
	}
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

func (t *tcp) callErrorListeners(e error, timestamp time.Time) error {
	rt := t.vu.Runtime()

	ev := t.newEvent(events.ERROR, timestamp)
	must(rt, ev.DefineDataProperty("error",
		rt.ToValue(e.Error()),
		sobek.FLAG_FALSE, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	for _, errorListener := range t.eventListeners.All(events.ERROR) {
		if _, err := errorListener(ev); err != nil { // TODO fix timestamp
			return err
		}
	}

	return nil
}

func (t *tcp) callOpenListeneres(timestamp time.Time) error {
	for _, openListener := range t.eventListeners.All(events.OPEN) {
		if _, err := openListener(t.newEvent(events.OPEN, timestamp)); err != nil {
			_ = t.conn.Close()                   // TODO log it?
			_ = t.connectionClosedWithError(err) // TODO log it?
			return err
		}
	}
	return nil
}

func (t *tcp) callEventListeners(evType string, timestamp time.Time) error {
	for _, listener := range t.eventListeners.All(evType) {
		if _, err := listener(t.newEvent(evType, timestamp)); err != nil {
			return err
		}
	}
	return nil
}

func (t *tcp) newEvent(eventType string, timestamp time.Time) *sobek.Object {
	rt := t.vu.Runtime()
	o := rt.NewObject()

	must(rt, o.DefineAccessorProperty("type", rt.ToValue(func() string {
		return eventType
	}), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	must(rt, o.DefineAccessorProperty("target", rt.ToValue(func() interface{} {
		return t.obj
	}), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE))
	// skip srcElement
	// skip currentTarget ??!!
	// skip eventPhase ??!!
	// skip stopPropagation
	// skip cancelBubble
	// skip stopImmediatePropagation
	// skip a bunch more

	must(rt, o.DefineAccessorProperty("timestamp", rt.ToValue(func() float64 {
		return float64(timestamp.UnixNano()) / 1_000_000 // milliseconds as double as per the spec
		// https://w3c.github.io/hr-time/#dom-domhighrestimestamp
	}), nil, sobek.FLAG_FALSE, sobek.FLAG_TRUE))

	return o
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
