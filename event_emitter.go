package eventemitter

import "sync"

type Listener func(interface{})

type ListenerType int

const (
	Once ListenerType = iota
	Always
)

type listenerWrapper struct {
	Id   int
	Func Listener
	Type ListenerType
}

type Event struct {
	Topic         string
	listeners     []listenerWrapper
	onceListeners []listenerWrapper
	idCounter     int
}

func (event *Event) nextId() int {
	event.idCounter++
	return event.idCounter
}

func (event *Event) add(id int, listener Listener, listenerType ListenerType) {
	wrapper := listenerWrapper{
		Id:   id,
		Func: listener,
		Type: listenerType,
	}
	if listenerType == Once {
		event.onceListeners = append(event.onceListeners, wrapper)
	} else {
		event.listeners = append(event.listeners, wrapper)
	}
}

func (event *Event) removeFromList(id int, listeners []listenerWrapper) []listenerWrapper {
	found := -1
	for index, listener := range listeners {
		if listener.Id == id {
			found = index
			break
		}
	}
	if found >= 0 {
		left := listeners[:found]
		return append(left, listeners[found+1:]...)
	}
	return listeners
}

func (event *Event) remove(id int) {
	event.onceListeners = event.removeFromList(id, event.onceListeners)
	event.listeners = event.removeFromList(id, event.listeners)
}

func (event *Event) clearOnceListeners() {
	event.onceListeners = make([]listenerWrapper, 0)
}

type newListener struct {
	Id       int
	Topic    string
	Listener Listener
	Type     ListenerType
}

type removedListener struct {
	Topic string
	Id    int
}

type incomingEvent struct {
	Topic   string
	Payload interface{}
}

type Emitter struct {
	Events          []*Event
	waitGroup       sync.WaitGroup
	newListenerId   chan int
	newListener     chan newListener
	removedListener chan removedListener
	stopManaging    chan bool
	incomingEvent   chan incomingEvent
	removedEvent    chan string
}

func NewEmitter() *Emitter {
	emitter := &Emitter{
		Events:          make([]*Event, 0),
		newListenerId:   make(chan int),
		newListener:     make(chan newListener, 1),
		removedListener: make(chan removedListener, 1),
		stopManaging:    make(chan bool, 1),
		incomingEvent:   make(chan incomingEvent),
		removedEvent:    make(chan string),
	}
	emitter.waitGroup.Add(1)
	go emitter.eventLoop()

	return emitter
}

func (emitter *Emitter) dispatch(listener listenerWrapper, payloadChannel chan interface{}) {
	payload := <-payloadChannel
	listener.Func(payload)
	emitter.waitGroup.Done()
}

func (emitter *Emitter) eventLoop() {
	shouldStop := false
	for {
		select {
		case incoming := <-emitter.newListener:
			{
				event := emitter.getEvent(incoming.Topic)
				if event == nil {
					event = emitter.newEvent(incoming.Topic)
				}
				id := event.nextId()
				event.add(id, incoming.Listener, incoming.Type)
				emitter.newListenerId <- id
			}
		case removed := <-emitter.removedListener:
			{
				event := emitter.getEvent(removed.Topic)
				if event != nil {
					event.remove(removed.Id)
				}
			}
		case incoming := <-emitter.incomingEvent:
			{
				event := emitter.getEvent(incoming.Topic)
				if event == nil {
					break
				}

				count := len(event.onceListeners) + len(event.listeners)
				payloadChannel := make(chan interface{})
				for _, listener := range event.onceListeners {
					emitter.waitGroup.Add(1)
					go emitter.dispatch(listener, payloadChannel)
				}

				event.clearOnceListeners()

				for _, listener := range event.listeners {
					emitter.waitGroup.Add(1)
					go emitter.dispatch(listener, payloadChannel)
				}

				for i := 0; i < count; i++ {
					payloadChannel <- incoming.Payload
				}
			}
		case topic := <-emitter.removedEvent:
			{
				found := -1
				for index, event := range emitter.Events {
					if event.Topic == topic {
						found = index
						break
					}
				}
				if found >= 0 {
					left := emitter.Events[:found]
					emitter.Events = append(left, emitter.Events[found+1:]...)
				}
			}

		case <-emitter.stopManaging:
			{
				shouldStop = true
				break
			}
		}

		if shouldStop {
			break
		}
	}
	emitter.waitGroup.Done()
}

func (emitter *Emitter) getEvent(topic string) *Event {
	for _, event := range emitter.Events {
		if event.Topic == topic {
			return event
		}
	}
	return nil
}

func (emitter *Emitter) newEvent(topic string) *Event {
	event := &Event{
		Topic:         topic,
		listeners:     make([]listenerWrapper, 0),
		onceListeners: make([]listenerWrapper, 0),
	}
	emitter.Events = append(emitter.Events, event)
	return event
}

func (emitter *Emitter) Once(topic string, listener Listener) int {
	emitter.newListener <- newListener{
		Topic:    topic,
		Listener: listener,
		Type:     Once,
	}
	id := <-emitter.newListenerId
	return id
}

func (emitter *Emitter) On(topic string, listener Listener) int {
	emitter.newListener <- newListener{
		Topic:    topic,
		Listener: listener,
		Type:     Always,
	}
	id := <-emitter.newListenerId
	return id
}

func (emitter *Emitter) Emit(topic string, payload interface{}) {
	emitter.incomingEvent <- incomingEvent{Topic: topic, Payload: payload}
}

func (emitter *Emitter) Topics() []string {
	topics := make([]string, 0)
	for _, event := range emitter.Events {
		topics = append(topics, event.Topic)
	}
	return topics
}

func (emitter *Emitter) Wait() {
	emitter.stopManaging <- true
	emitter.waitGroup.Wait()
}

func (emitter *Emitter) Off(topic string, ids ...int) {
	if len(ids) > 0 {
		for _, id := range ids {
			emitter.removedListener <- removedListener{
				Topic: topic,
				Id:    id,
			}
		}
	} else {
		emitter.removedEvent <- topic
	}
}
