package ratchet

import (
	math_rand "math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/andres-erbsen/chatterbox/ratchet/proto"
)

func TestRatchetFaceSelfTrip(t *testing.T) {
	popr := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	p := proto.NewPopulatedRatchetState(popr, true)
	msg := NewRatchetFromFace(p)
	msg2 := NewRatchetFromFace(msg)
	if !reflect.DeepEqual(msg.saved, msg2.saved) {
		t.Fatalf("%#v !Face Equal by .saved %#v", msg.saved, msg2.saved)
	}
	if !reflect.DeepEqual(msg, msg2) {
		t.Fatalf("%#v !Face Equal %#v", msg, msg2)
	}
}

func TestRatchetFaceRoundTrip(t *testing.T) {
	popr := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	p := proto.NewPopulatedRatchetState(popr, true)
	msg := NewRatchetFromFace(p)
	msgRoundTrip := NewRatchetFromFace(proto.NewRatchetStateFromFace(msg))
	if !reflect.DeepEqual(msg.saved, msgRoundTrip.saved) {
		t.Fatalf("%#v !Face Equal by .saved %#v", msg.saved, msgRoundTrip.saved)
	}
	if !reflect.DeepEqual(msg, msgRoundTrip) {
		t.Fatalf("%#v !Face Equal %#v", msg, msgRoundTrip)
	}
}
