package log

import "fmt"

func Info(format string, a ...interface{}) {
	fmt.Printf("INFO:  "+format+"\n", a...)
}

func Debug(format string, a ...interface{}) {
	fmt.Printf("DEBUG: "+format+"\n", a...)
}

func Panic(format string, a ...interface{}) {
	fmt.Printf("PANIC: "+format+"\n", a...)
	panic(fmt.Errorf(format, a...))
}
