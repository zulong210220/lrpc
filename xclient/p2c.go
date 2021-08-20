package xclient

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// https://github.com/hnlq715/go-loadbalance

type peakEwma struct {
	stamp int64
	value int64
	tau   time.Duration
}

const (
	defaultTau = 10000 * time.Millisecond
)

func newPEWMA() *peakEwma {
	return &peakEwma{
		tau: defaultTau,
	}
}

// Observe 计算peak指数加权移动平均值
func (p *peakEwma) Observe(rtt int64) {
	if p == nil {
		return
	}
	now := time.Now().UnixNano()

	stamp := atomic.SwapInt64(&p.stamp, now)
	td := now - stamp

	if td < 0 {
		td = 0
	}

	w := math.Exp(float64(-td) / float64(p.tau))
	latency := atomic.LoadInt64(&p.value)

	if rtt > latency {
		atomic.StoreInt64(&p.value, rtt)
	} else {
		atomic.StoreInt64(&p.value, int64(float64(latency)*w+float64(rtt)*(1.0-w)))
	}
}

func (p *peakEwma) Value() int64 {
	return atomic.LoadInt64(&p.value)
}

type peakEwmaNode struct {
	item    string
	latency *peakEwma
	weight  float64
}

type pewma struct {
	items []*peakEwmaNode
	mu    sync.Mutex
	rand  *rand.Rand
}
