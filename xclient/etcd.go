package xclient

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/zulong210220/lrpc/consts"
	"github.com/zulong210220/lrpc/log"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EtcdDiscovery struct {
	client   *clientv3.Client
	mu       sync.RWMutex
	services map[string][]string
	r        *rand.Rand
	index    int
	p2cs     map[string][]*peakEwmaNode
	edp2c    map[string]*peakEwma // ip:port=>latency
}

func NewEtcdDiscovery(ea []string, et int, ss []string) *EtcdDiscovery {
	ed := &EtcdDiscovery{
		services: make(map[string][]string),
		r:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	config := clientv3.Config{
		Endpoints:   ea,
		DialTimeout: time.Duration(et) * time.Second,
	}
	var err error
	ed.client, err = clientv3.New(config)
	if err != nil {
		log.Errorf("", "NewEtcdDiscovery err:", err)
		return ed
	}

	go ed.WatchServices(ss)

	return ed
}

func (ed *EtcdDiscovery) WatchServices(ss []string) error {
	for _, s := range ss {
		ed.watchService(s)
	}
	return nil
}

func (ed *EtcdDiscovery) watchService(s string) {
	fun := "EtcdDiscovery.watchService"
	path := ed.GetRegPath(s)
	resp, err := ed.client.Get(context.Background(), path, clientv3.WithPrefix())
	if err != nil {
		log.Errorf("", "%s client.Get path:%s err:%v", fun, path, err)
		return
	}
	log.Infof("", "watchServices %s %s resp:%+v", path, s, resp)

	ed.mu.Lock()
	defer ed.mu.Unlock()

	if ed.services[s] == nil {
		ed.services[s] = []string{}
	}

	//遍历获取到的key和value
	for _, ev := range resp.Kvs {
		ed.services[s] = append(ed.services[s], getEndpointFromKey(string(ev.Key)))
	}

	//监视前缀，修改变更的server
	go ed.watcher(path)
}

func (ed *EtcdDiscovery) GetRegPath(s string) string {
	return fmt.Sprintf("%s/%s", consts.DefaultRegPath, s)
}

// ll/kk/app/ip:port
func (ed *EtcdDiscovery) watcher(path string) {
	rch := ed.client.Watch(context.Background(), path, clientv3.WithPrefix())
	log.Infof("", "watching prefix:%s now...", path)

	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Infof("etcd watch", "wresp events %+v", ev)
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				ed.SetServices(path, getEndpointFromKey(string(ev.Kv.Key)))
			case mvccpb.DELETE: //删除
				ed.DelServices(path, getEndpointFromKey(string(ev.Kv.Key)))
			}
		}
	}
	log.Infof("", "watching prefix:%s ending...", path)
}

// service name
func getSNFromPath(path string) string {
	ss := strings.Split(path, "/")
	if len(ss) < 2 {
		return ""
	}
	return ss[len(ss)-1]
}

func getEndpointFromKey(key string) string {
	ss := strings.Split(key, "/")
	if len(ss) < 2 {
		return ""
	}
	return ss[len(ss)-1]
}

func (ed *EtcdDiscovery) SetServices(path, endpoint string) {
	log.Infof("etcd", "Set services path:%s endpoint:%s", path, endpoint)
	// TODO
	s := getSNFromPath(path)
	ed.mu.Lock()
	defer ed.mu.Unlock()

	log.Infof("etcd", "services:%+v, before %s", ed.services[s], s)
	if ed.services[s] == nil {
		ed.services[s] = []string{endpoint}
		return
	}

	exist := false
	for _, e := range ed.services[s] {
		if e == endpoint {
			exist = true
		}
	}

	log.Infof("etcd before", "services:%+v, s:%s exist:%v", ed.services[s], s, exist)
	if !exist {
		ed.services[s] = append(ed.services[s], endpoint)
	}
	log.Infof("etcd after", "services:%+v, s:%s exist:%v", ed.services[s], s, exist)
	return
}

func (ed *EtcdDiscovery) DelServices(path, endpoint string) {
	log.Infof("etcd", "Del services path:%s endpoint:%s", path, endpoint)
	// TODO
	s := getSNFromPath(path)
	ed.mu.Lock()
	defer ed.mu.Unlock()

	if ed.services[s] == nil {
		return
	}

	log.Infof("etcd before", "Del services path:%s endpoint:%s servers:%+v", path, endpoint, ed.services[s])
	es := []string{}
	for _, e := range ed.services[s] {
		if e == endpoint {
			continue
		}
		es = append(es, e)
	}

	ed.services[s] = es

	log.Infof("etcd after", "Del services path:%s endpoint:%s servers:%+v", path, endpoint, ed.services[s])
	return
}

func (ed *EtcdDiscovery) Get(sn string, sm SelectMode) (string, error) {
	if sm == P2cSelect {
		return ed.p2cNext(sn)
	}

	ed.mu.Lock()
	defer ed.mu.Unlock()

	services := ed.services[sn]
	n := len(services)
	if n == 0 {
		res, err := ed.GetService(sn)
		if err != nil {
			log.Errorf("", "Etcd Discovery GetService %s failed err:%v", sn, err)
			return "", err
		}
		if len(res) == 0 {
			return "", errors.New("rpc discovery: no avaialable servers")
		}
		ed.services[sn] = res
		services = res
	}

	switch sm {
	case RandomSelect:
		return services[ed.r.Intn(n)], nil
	case RoundRobinSelect:
		s := services[ed.index%n]
		ed.index = (ed.index + 1) % n
		return s, nil
	default:
		log.Error("", "invalid select mode ", sm)
		return "", errors.New("rpc discovery: not supported SelectMode")
	}
}

func (ed *EtcdDiscovery) GetAll(sn string) ([]string, error) {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	n := len(ed.services[sn])
	ss := make([]string, n)
	copy(ss, ed.services[sn])
	return ss, nil
}

func (ed *EtcdDiscovery) GetService(sn string) ([]string, error) {
	fun := "EtcdDiscovery.GetService"
	ctx := context.Background()
	res := []string{}

	log.Infof("", "%s before get ...sn:%s", fun, sn)
	key := ed.GetRegPath(sn)
	resp, err := ed.client.Get(ctx, key, clientv3.WithPrevKV())
	log.Infof("", "%s get %s...resp:%+v err:%v", fun, key, resp, err)
	if err != nil {
		log.Errorf("%s client get key:%s err:%v", fun, key, err)
		return res, err
	}

	if resp.Count <= 0 {
		return res, err
	}
	log.Infof("", "%s get %+v", fun, resp)
	for k, _ := range resp.Kvs {
		res = append(res, getEndpointFromKey(string(k)))
	}

	return res, err
}

func (ed *EtcdDiscovery) p2cNext(sn string) (string, error) {
	ss := ed.p2cs[sn]
	n := len(ss)

	ed.mu.Lock()
	defer ed.mu.Unlock()
	fmt.Println("---- ", n)
	if n == 0 {
		res, err := ed.GetService(sn)
		if err != nil {
			log.Errorf("", "Etcd Discovery GetService %s failed err:%v", sn, err)
			return "", err
		}
		fmt.Println(sn, res)
		if len(res) == 0 {
			return "", errors.New("rpc discovery: no avaialable servers")
		}
		sss := []*peakEwmaNode{}

		for _, s := range res {
			sss = append(ss, &peakEwmaNode{item: s, latency: newPEWMA(), weight: 1})
		}
		ed.p2cs[sn] = sss
		ss = sss
	}

	n = len(ss)
	if n == 1 {
		return ss[0].item, nil
	}

	a := ed.r.Intn(len(ss))
	b := ed.r.Intn(len(ss) - 1)

	if b >= a {
		b++
	}

	sc, backsc := ss[a], ss[b]

	// choose the least loaded item based on inflight and weight
	if float64(sc.latency.Value())*backsc.weight > float64(backsc.latency.Value())*sc.weight {
		sc, backsc = backsc, sc
	}
	if ed.edp2c[sc.item] == nil {
		ed.edp2c[sc.item] = sc.latency
	}

	return sc.item, nil
}

func (ed *EtcdDiscovery) Observe(rpcAddr string, dur int64) {
	if rpcAddr == "" {
		return
	}
	ss := strings.Split(rpcAddr, "@")
	n := len(ss)
	endpoint := ""
	if n == 1 {
		endpoint = ss[0]
	}
	endpoint = ss[1]

	ed.edp2c[endpoint].Observe(dur)
}
