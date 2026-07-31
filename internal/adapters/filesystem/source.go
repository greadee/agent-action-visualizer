package filesystem

import (
	"sync"

	"github.com/fsnotify/fsnotify"
)

type operation uint8

const (
	opCreate operation = 1 << iota
	opWrite
	opRemove
	opRename
	opChmod
)

type change struct {
	path string
	op   operation
}

type eventSource interface {
	Add(string) error
	Close() error
	Events() <-chan change
	Errors() <-chan error
}

type fsnotifySource struct {
	watcher *fsnotify.Watcher
	events  chan change
	done    chan struct{}
	closing chan struct{}
	once    sync.Once
}

func newFSNotifySource(queueSize int) (eventSource, error) {
	watcher, err := fsnotify.NewBufferedWatcher(uint(queueSize))
	if err != nil {
		return nil, err
	}
	source := &fsnotifySource{
		watcher: watcher,
		events:  make(chan change, queueSize),
		done:    make(chan struct{}),
		closing: make(chan struct{}),
	}
	go source.forward()
	return source, nil
}

func (s *fsnotifySource) Add(path string) error {
	return s.watcher.Add(path)
}

func (s *fsnotifySource) Close() error {
	s.once.Do(func() { close(s.closing) })
	err := s.watcher.Close()
	<-s.done
	return err
}

func (s *fsnotifySource) Events() <-chan change {
	return s.events
}

func (s *fsnotifySource) Errors() <-chan error {
	return s.watcher.Errors
}

func (s *fsnotifySource) forward() {
	defer close(s.done)
	defer close(s.events)
	for event := range s.watcher.Events {
		var op operation
		if event.Has(fsnotify.Create) {
			op |= opCreate
		}
		if event.Has(fsnotify.Write) {
			op |= opWrite
		}
		if event.Has(fsnotify.Remove) {
			op |= opRemove
		}
		if event.Has(fsnotify.Rename) {
			op |= opRename
		}
		if event.Has(fsnotify.Chmod) {
			op |= opChmod
		}
		if op != 0 {
			select {
			case s.events <- change{path: event.Name, op: op}:
			case <-s.closing:
				return
			}
		}
	}
}
