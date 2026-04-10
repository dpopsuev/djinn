package discourse

import "sync"

var _ Forum = (*StubForum)(nil)

// StubForum implements Forum for testing. Stores threads in memory.
type StubForum struct {
	mu      sync.Mutex
	threads map[ThreadID]*Thread
	nextID  int
}

func NewStubForum() *StubForum {
	return &StubForum{threads: make(map[ThreadID]*Thread)}
}

func (f *StubForum) Post(topic, message string) (ThreadID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	id := ThreadID(topic + "-" + string(rune('0'+f.nextID)))
	f.threads[id] = &Thread{ID: id, Topic: topic, Messages: []string{message}}
	return id, nil
}

func (f *StubForum) Reply(id ThreadID, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.threads[id]; ok {
		t.Messages = append(t.Messages, message)
	}
	return nil
}

func (f *StubForum) Threads(topic string) []Thread {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Thread
	for _, t := range f.threads {
		if t.Topic == topic {
			out = append(out, *t)
		}
	}
	return out
}
