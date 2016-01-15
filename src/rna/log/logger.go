package log

import "fmt"

func Info(msg string) {
	fmt.Printf("Info: %s\n", msg)
}

func Debug(msg string) {
	fmt.Printf("Debug: %s\n", msg)
}

func Panic(msg error) {
	fmt.Printf("PANIC: %v\n", msg)
	panic(msg)
}
