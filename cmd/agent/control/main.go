package main

/*
#cgo LDFLAGS: -lkubeguard
#include <stdlib.h>
#include <kubeguard/kubeguardlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func main() {
	path := "/example.conf"
	rulesPath := C.CString(path)
	defer C.free(unsafe.Pointer(rulesPath))

	fmt.Print("Hello world!")
	x := 42
	C.dump_rules(rulesPath)
	fmt.Printf("The value of x is: %d\n", x)
}
