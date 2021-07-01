package core

import (
	"fmt"
	"log"
	"os"
)

var (
	Debug bool = true
)

// Printf formats according to a format specifier and writes to standard output.
// It returns the number of bytes written and any write error encountered.
func Printf(format string, a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprintf(os.Stdout, format, a...)
}

// Print formats using the default formats for its operands and writes to standard output.
// Spaces are added between operands when neither is a string.
// It returns the number of bytes written and any write error encountered.
func Print(a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprint(os.Stdout, a...)
}

// Println formats using the default formats for its operands and writes to standard output.
// Spaces are always added between operands and a newline is appended.
// It returns the number of bytes written and any write error encountered
func Println(a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprintln(os.Stdout, a...)
}
func mytest() {
	Println("fff", Debug)
}

type ThreadTask struct {
	Fn   func(int, interface{})
	Argv interface{}
}

type ThreadPool struct {
	taskQueue chan *ThreadTask
	stop      chan int
	Size      int
}

func tryCatchException(id int, task *ThreadTask) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("execute task to failed,err = ", err)
		}
	}()
	task.Fn(id, task.Argv)
}
func genWorker(id int, pool *ThreadPool) {
	for {
		task, more := <-pool.taskQueue
		if more {
			tryCatchException(id, task)
		} else {
			break
		}
	}
	pool.stop <- 1
}

func NewThreadPool(size int) *ThreadPool {
	pool := new(ThreadPool)
	pool.taskQueue = make(chan *ThreadTask, 10)
	pool.stop = make(chan int)
	if size < 1 {
		size = 1
	}
	pool.Size = size
	for i := 0; i < pool.Size; i++ {
		go genWorker(i, pool)
	}
	return pool
}
func (pool *ThreadPool) Execute(task *ThreadTask) bool {
	if task == nil {
		return false
	}
	pool.taskQueue <- task
	return true
}
func (pool *ThreadPool) Shutdown() {
	close(pool.taskQueue)
	for i := 0; i < pool.Size; i++ {
		<-pool.stop
	}
}
func callFn(ch chan int, fn func(interface{}), arg interface{}) {
	fn(arg)
	ch <- 1
}
func ExecuteFn(fn func(interface{}), list []interface{}) {
	if list == nil || fn == nil {
		return
	}
	ch := make(chan int)
	for _, arg := range list {
		go callFn(ch, fn, arg)
	}
	liLen := len(list)
	for i := 0; i < liLen; i++ {
		<-ch
	}
}

func ExistFile(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}
