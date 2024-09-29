package dryrun

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
)

type Recorder interface {
	Record(object runtime.Object, reason, message string)
	Recordf(object runtime.Object, reason, messageFmt string, args ...interface{})
}

type SimpleRecorder struct{}

func (r *SimpleRecorder) Record(object runtime.Object, reason, message string) {
	fmt.Println("#######")
	fmt.Println("object", object)
	fmt.Println("reason", reason)
	fmt.Println("message", message)
	fmt.Println("#######")
}

func (r *SimpleRecorder) Recordf(object runtime.Object, reason, messageFmt string, args ...interface{}) {
	fmt.Println("#######")
	fmt.Println("object", object)
	fmt.Println("reason", reason)
	fmt.Printf(messageFmt+"\n", args...)
	fmt.Println("#######")
}
