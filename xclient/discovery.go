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
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/log"
)

type SelectMode int

const (
	RandomSelect SelectMode = iota + 1
	RoundRobinSelect
	P2cSelect
)

type Discovery interface {
	Get(sn string, mode SelectMode) (string, error)
	GetAll(sn string) ([]string, error)
	Observe(rpaAddr string, dur int64)
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

func (md *MultiServersDiscovery) Observe(rpcAddr string, dur int64) {

}

func (d *MultiServersDiscovery) Refresh() error {
	return nil
}

func (d *MultiServersDiscovery) Update(sn string, ss []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = ss
	return nil
}

func (d *MultiServersDiscovery) Get(sn string, mode SelectMode) (string, error) {
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

func (d *MultiServersDiscovery) GetAll(sn string) ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ss := make([]string, len(d.servers), len(d.servers))
	copy(ss, d.servers)
	return ss, nil
}

type RegistryDiscovery struct {
	*MultiServersDiscovery
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

func NewRegistryDiscovery(ra string, to time.Duration) *RegistryDiscovery {
	if to == 0 {
		to = consts.DefaultUpdateTimeout
	}

	d := &RegistryDiscovery{
		MultiServersDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:              ra,
		timeout:               to,
	}
	return d
}

func (rd *RegistryDiscovery) Update(ss []string) error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	rd.servers = ss
	rd.lastUpdate = time.Now()

	return nil
}

func (rd *RegistryDiscovery) Refresh() error {
	fun := "RegistryDiscovery.Refresh"
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.lastUpdate.Add(rd.timeout).After(time.Now()) {
		return nil
	}

	resp, err := http.Get(rd.registry)
	if err != nil {
		log.Errorf("", "%s rpc registry refresh err:%v", fun, err)
		return err
	}

	ss := strings.Split(resp.Header.Get("X-lrpc-servers"), ",")
	rd.servers = make([]string, 0, len(ss))

	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s != "" {
			rd.servers = append(rd.servers, s)
		}
	}

	rd.lastUpdate = time.Now()
	return nil
}

func (rd *RegistryDiscovery) Get(sm SelectMode) (string, error) {
	err := rd.Refresh()
	if err != nil {
		return "", err
	}
	sn := ""
	return rd.MultiServersDiscovery.Get(sn, sm)
}

func (rd *RegistryDiscovery) GetAll() ([]string, error) {
	err := rd.Refresh()
	if err != nil {
		return nil, err
	}

	sn := ""
	return rd.MultiServersDiscovery.GetAll(sn)
}

/* vim: set tabstop=4 set shiftwidth=4 */
