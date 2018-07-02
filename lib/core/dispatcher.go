package core

type Dispatcher interface {
	Subscribe(string, Callback)
	SubscribeID(string, interface{}, Callback)
	UnsubscribeID(string, interface{}) int
	Dispatch(string, interface{}) bool
	ClearSubscriptions()
	CancelDispatch()
}

type Dsptchr struct {
	evmap  map[string][]subscription // maps event name to subcriptions list
	cancel bool                      // flag informing cancelled dispatch
}

type Callback func(string, interface{})

type subscription struct {
	id interface{}
	cb func(string, interface{})
}

// NewEventDispatcher creates and returns a pointer to an Event Dispatcher
func NewDispatcher() *Dsptchr {
	ed := new(Dsptchr)
	ed.Initialize()
	return ed
}

// Initialize initializes this event dispatcher.
// It is normally used by other types which embed an event dispatcher
func (d *Dsptchr) Initialize() {
	d.evmap = make(map[string][]subscription)
}

// Subscribe subscribes to receive events with the given name.
// If it is necessary to unsubscribe the event, the function SubscribeID
// should be used.
func (d *Dsptchr) Subscribe(evname string, cb Callback) {
	d.SubscribeID(evname, nil, cb)
}

// Subscribe subscribes to receive events with the given name.
// The function accepts a unique id to be use to unsubscribe this event
func (d *Dsptchr) SubscribeID(evname string, id interface{}, cb Callback) {
	d.evmap[evname] = append(d.evmap[evname], subscription{id, cb})
}

// Unsubscribe unsubscribes from the specified event and subscription id
// Returns the number of subscriptions found.
func (d *Dsptchr) UnsubscribeID(evname string, id interface{}) int {
	// Get list of subscribers for this event
	// If not found, nothing to do
	subs, ok := d.evmap[evname]
	if !ok {
		return 0
	}

	// Remove all subscribers with the specified id for this event
	found := 0
	pos := 0
	for pos < len(subs) {
		if subs[pos].id == id {
			copy(subs[pos:], subs[pos+1:])
			subs[len(subs)-1] = subscription{}
			subs = subs[:len(subs)-1]
			found++
		} else {
			pos++
		}
	}
	//log.Debug("Dispatcher(%p).UnsubscribeID:%s (%p): %v",ed, evname, id, found)
	d.evmap[evname] = subs
	return found
}

// Dispatch dispatch the specified event and data to all registered subscribers.
// The function returns true if the propagation was cancelled by a subscriber.
func (d *Dsptchr) Dispatch(evname string, ev interface{}) bool {
	// Get list of subscribers for this event
	subs := d.evmap[evname]
	if subs == nil {
		return false
	}

	// Dispatch to all subscribers
	d.cancel = false
	for i := 0; i < len(subs); i++ {
		subs[i].cb(evname, ev)
		if d.cancel {
			break
		}
	}
	return d.cancel
}

// ClearSubscriptions clear all subscriptions from this dispatcher
func (d *Dsptchr) ClearSubscriptions() {
	d.evmap = make(map[string][]subscription)
}

// CancelDispatch cancels the propagation of the current event.
// No more subscribers will be called for this event dispatch.
func (d *Dsptchr) CancelDispatch() {
	d.cancel = true
}
