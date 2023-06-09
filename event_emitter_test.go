package eventemitter_test

import (
	"testing"

	eventemitter "github.com/asif-mahmud/event-emitter"
)

func TestCanCreateNewEmitter(t *testing.T) {
	emitter := eventemitter.NewEmitter()
	if emitter == nil {
		t.Errorf("Failed to create new Emitter instance")
	}
}

func TestInitiallyEmptyEmitter(t *testing.T) {
	emitter := eventemitter.NewEmitter()
	topics := emitter.Topics()
	if len(topics) > 0 {
		t.Errorf("Newly created emitter should not have any topic")
	}
}
