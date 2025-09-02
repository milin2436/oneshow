package core

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	Debug bool = true
)

type Worker struct {
	wid int
}
type Task interface {
	GetTaskID() int
	Execute(w *Worker) error
}

type BaseTask struct {
	id   int
	task func(w *Worker) error
}

type TaskManager struct {
	taskBufferQueue chan Task
	workerMsg       chan int

	workerMaxSize    int
	activeWorkerSize int

	sequenceWorkerID int
	taskList         *DoublyLinkedList
}

type Node struct {
	task Task
	prev *Node
	next *Node
}

type DoublyLinkedList struct {
	size int
	head *Node
	tail *Node
}

func (list *DoublyLinkedList) GetHead() Task {
	if list.head == nil {
		return nil
	}
	return list.head.task
}
func (list *DoublyLinkedList) Append(t Task) bool {
	if t == nil {
		return false
	}
	newNode := &Node{task: t}
	if list.tail == nil {
		list.head = newNode
		list.tail = newNode
	} else {
		list.tail.next = newNode
		newNode.prev = list.tail
		list.tail = newNode
	}
	list.size++
	return true
}
func (list *DoublyLinkedList) DelHead() bool {
	Printf("task SIZE %d\n", list.size)
	if list.head == nil {
		return false
	}
	if list.size == 1 {
		list.head = nil
		list.tail = nil
	} else {
		curHead := list.head

		newHead := list.head.next
		newHead.prev = nil
		list.head = newHead

		curHead.next = nil
		curHead.task = nil
	}
	list.size--
	return true
}

func (list *DoublyLinkedList) Delete(node *Node) bool {
	if node == nil {
		return false
	}
	Printf("task SIZE %d\n", list.size)
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		list.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		list.tail = node.prev
	}
	list.size--
	return true
}

func (list *DoublyLinkedList) Traverse() {
	for current := list.head; current != nil; current = current.next {
		fmt.Println(current.task)
	}
}

func (t *BaseTask) GetTaskID() int {
	return t.id
}
func (t *BaseTask) Execute(w *Worker) error {
	if t.task != nil {
		return t.task(w)
	}
	return nil
}
func (t *BaseTask) SetTask(task func(*Worker) error) {
	t.task = task
}

func NewTaskManager() *TaskManager {
	tm := new(TaskManager)
	tm.workerMaxSize = 1
	tm.taskBufferQueue = make(chan Task, 1024)
	tm.workerMsg = make(chan int, 64)
	tm.taskList = new(DoublyLinkedList)
	return tm
}

func (tm *TaskManager) SetActiveWorkerMaxSize(s int) {
	tm.workerMaxSize = s
}
func (tm *TaskManager) AddTask(t Task) {
	tm.taskBufferQueue <- t
}
func (tm *TaskManager) NewWorker() *Worker {
	w := new(Worker)
	tm.sequenceWorkerID++
	w.wid = tm.sequenceWorkerID
	return w
}

func (tm *TaskManager) getNextTask() *Node {
	t := tm.checkTasksInQueue()
	if t != nil {
		tm.taskList.Append(t)
	}
	return tm.taskList.head
}
func (tm *TaskManager) checkTasksInQueue() Task {
	var t Task
	select {
	case t = <-tm.taskBufferQueue:
		Printf("Received task ,id =  %d\n", t.GetTaskID())
		return t
	default:
		return nil
	}
}
func (tm *TaskManager) updateActiveWorkerSize() {
	var wid int
	for {
		select {
		case wid = <-tm.workerMsg:
			Printf("Worker completed, Worker ID  =  %d\n", wid)
			tm.activeWorkerSize--
		default:
			return
		}
	}
}

func (tm *TaskManager) start(t Task) {
	w := tm.NewWorker()
	defer func() {
		if err := recover(); err != nil {
			log.Println("execute task to failed,err = ", err)
			tm.workerMsg <- w.wid
		}
	}()
	Printf("execute task, wid = %d ,task id = %d \n", w.wid, t.GetTaskID())
	err := t.Execute(w)
	if err != nil {
		Printf("err = %s\n", err.Error())
	}
	tm.workerMsg <- w.wid
}

func (tm *TaskManager) executeTask(t Task) bool {
	if tm.activeWorkerSize < tm.workerMaxSize {
		tm.activeWorkerSize++
		go tm.start(t)
		return true
	}
	return false
}

const wait_time = 200 * time.Millisecond

func (tm *TaskManager) Wait4Completion() {
	for {
		t := tm.getNextTask()
		if t != nil {
			//Make every effort to acquire an available worker
			tm.updateActiveWorkerSize()
			if tm.executeTask(t.task) {
				//tm.taskList.DelHead()
				tm.taskList.Delete(t)
				continue
			} else {
				//wait a worker
				time.Sleep(wait_time)
			}
		} else {
			tm.updateActiveWorkerSize()
			if t == nil && tm.activeWorkerSize == 0 {
				break
			} else {
				//tasklist is null， wait worker completion
				time.Sleep(wait_time)
			}
		}
	}
	Printf("task manager normal exit...\n")
}

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
	Println("flag value ", Debug)
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
	return err == nil
}
