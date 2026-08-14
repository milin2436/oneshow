package core

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Debug enables debug output from Debugf/DebugPrint/DebugPrintln.
var Debug bool = true

type Worker struct {
	wid int
}

type Task interface {
	TaskID() int
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

func (list *DoublyLinkedList) Delete(node *Node) bool {
	if node == nil {
		return false
	}
	Debugf("task SIZE %d\n", list.size)
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

func (t *BaseTask) TaskID() int {
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

func (tm *TaskManager) SetWorkerMaxSize(s int) {
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
		Debugf("Received task ,id =  %d\n", t.TaskID())
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
			Debugf("Worker completed, Worker ID  =  %d\n", wid)
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
	Debugf("execute task, wid = %d ,task id = %d \n", w.wid, t.TaskID())
	err := t.Execute(w)
	if err != nil {
		Debugf("err = %s\n", err.Error())
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

const waitTime = 200 * time.Millisecond

// Wait blocks until all queued tasks have been executed.
func (tm *TaskManager) Wait() {
	for {
		t := tm.getNextTask()
		if t != nil {
			//Make every effort to acquire an available worker
			tm.updateActiveWorkerSize()
			if tm.executeTask(t.task) {
				tm.taskList.Delete(t)
				continue
			} else {
				//wait a worker
				time.Sleep(waitTime)
			}
		} else {
			tm.updateActiveWorkerSize()
			if t == nil && tm.activeWorkerSize == 0 {
				break
			} else {
				//tasklist is null， wait worker completion
				time.Sleep(waitTime)
			}
		}
	}
	Debugf("task manager normal exit...\n")
}

// Debugf formats according to a format specifier and writes to standard output
// when Debug is enabled.
func Debugf(format string, a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprintf(os.Stdout, format, a...)
}

// DebugPrint formats using the default formats for its operands and writes to
// standard output when Debug is enabled.
func DebugPrint(a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprint(os.Stdout, a...)
}

// DebugPrintln formats using the default formats for its operands, appends a
// newline, and writes to standard output when Debug is enabled.
func DebugPrintln(a ...interface{}) (n int, err error) {
	if !Debug {
		return 0, nil
	}
	return fmt.Fprintln(os.Stdout, a...)
}

// ExistFile reports whether the file at path exists.
func ExistFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
