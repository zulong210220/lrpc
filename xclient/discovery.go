package xclient

/*
 * Author : lijinya
 * Email : yajin160305@gmail.com
 * File : discovery.go
 * CreateDate : 2021-07-20 18:44:36
 * */
import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect SelectMode = iota + 1
	RoundRobinSelect
)

type Discovery interface {
	Refresh() error
	Update(ss []string) error
	Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
}

type MultiServersDiscovery struct {
	r       *rand.Rand
	mu      sync.RWMutex
	servers []string
	index   int
}

func NewMultiServerDiscovery(ss []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: ss,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

var (
	_ Discovery = (*MultiServersDiscovery)(nil)
)

func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(ss []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = ss
	return nil
}

func (d *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no avaialable servers")
	}

	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n]
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported SelectMode")
	}
}

func (d *MultiServersDiscovery) GetAll() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ss := make([]string, len(d.servers), len(d.servers))
	copy(ss, d.servers)
	return ss, nil
}

/* vim: set tabstop=4 set shiftwidth=4 */
