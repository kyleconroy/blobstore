package blobstore

import "reflect"

type Tester interface {
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
}

func Equal(t Tester, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected `%v` (type %v) - Got `%v` (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func NotEqual(t Tester, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		t.Errorf("Expected `%v` (type %v) - Got `%v` (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}
