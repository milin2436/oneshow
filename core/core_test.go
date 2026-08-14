package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoublyLinkedListAppendDelete(t *testing.T) {
	list := new(DoublyLinkedList)
	task1 := &BaseTask{id: 1}

	if list.Append(nil) {
		t.Fatal("Append(nil) should return false")
	}
	if !list.Append(task1) {
		t.Fatal("Append should return true")
	}
	if list.size != 1 || list.head == nil || list.head.task != task1 {
		t.Fatal("head not set correctly after first Append")
	}
	if list.tail == nil || list.tail.task != task1 {
		t.Fatal("tail not set correctly after first Append")
	}

	task2 := &BaseTask{id: 2}
	list.Append(task2)
	if list.size != 2 {
		t.Fatalf("size = %d, want 2", list.size)
	}

	// Delete the head: task1 out, task2 becomes head
	if !list.Delete(list.head) {
		t.Fatal("Delete should return true")
	}
	if list.size != 1 || list.head.task != task2 {
		t.Fatal("head should now be task2")
	}
	if list.head.prev != nil {
		t.Fatal("new head prev should be nil")
	}

	// Delete the last remaining node
	if !list.Delete(list.head) {
		t.Fatal("Delete last should return true")
	}
	if list.size != 0 || list.head != nil || list.tail != nil {
		t.Fatal("empty list should have nil head/tail and size 0")
	}
	if list.Delete(nil) {
		t.Fatal("Delete(nil) should return false")
	}
}

func TestBaseTaskTaskID(t *testing.T) {
	bt := &BaseTask{id: 42}
	if bt.TaskID() != 42 {
		t.Fatalf("TaskID() = %d, want 42", bt.TaskID())
	}
}

func TestTaskManagerRunsAllTasks(t *testing.T) {
	tm := NewTaskManager()
	tm.SetWorkerMaxSize(2)
	ran := make(chan int, 10)
	for i := 0; i < 5; i++ {
		t := &BaseTask{id: i}
		t.SetTask(func(*Worker) error {
			ran <- 1
			return nil
		})
		tm.AddTask(t)
	}
	tm.Wait()
	if got := len(ran); got != 5 {
		t.Fatalf("executed %d tasks, want 5", got)
	}
}

func TestExistFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if ExistFile(p) {
		t.Fatal("missing file should not exist")
	}
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !ExistFile(p) {
		t.Fatal("existing file should exist")
	}
}
