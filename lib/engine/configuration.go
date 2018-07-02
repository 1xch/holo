package engine

import (
	"os"
	"sort"

	"github.com/Laughs-In-Flowers/holo/lib/core"
	"github.com/Laughs-In-Flowers/log"
)

type ConfigFn func(*Engine) error

type Config interface {
	Order() int
	Configure(*Engine) error
}

type config struct {
	order int
	fn    ConfigFn
}

func DefaultConfig(fn ConfigFn) Config {
	return config{50, fn}
}

func NewConfig(order int, fn ConfigFn) Config {
	return config{order, fn}
}

func (c config) Order() int {
	return c.order
}

func (c config) Configure(e *Engine) error {
	return c.fn(e)
}

type configList []Config

func (c configList) Len() int { return len(c) }

func (c configList) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c configList) Less(i, j int) bool { return c[i].Order() < c[j].Order() }

type Configuration struct {
	e          *Engine
	configured bool
	list       configList
}

func (c *Configuration) Init(e *Engine, conf ...Config) {
	c.e = e
	c.list = builtIns
	c.Add(conf...)
}

func (c *Configuration) Add(conf ...Config) {
	c.list = append(c.list, conf...)
}

func (c *Configuration) AddFn(fns ...ConfigFn) {
	for _, fn := range fns {
		c.list = append(c.list, DefaultConfig(fn))
	}
}

func configure(e *Engine, conf ...Config) error {
	for _, c := range conf {
		err := c.Configure(e)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Configuration) Configure() error {
	sort.Sort(c.list)

	err := configure(c.e, c.list...)
	if err == nil {
		c.configured = true
	}

	return err
}

func (c *Configuration) Configured() bool {
	return c.configured
}

var builtIns = []Config{
	config{001, eState},
	config{002, eDefaults},
	config{101, eLogger},
	config{102, eError},
	config{501, eWorld},
	config{601, eInner},
}

func eState(e *Engine) error {
	e.State = newState()
	return nil
}

func eDefaults(e *Engine) error {
	e.resetSettings()
	e.resetState()
	return nil
}

func SetDebug(b bool) Config {
	return NewConfig(50,
		func(e *Engine) error {
			e.debug = b
			return nil
		})
}

func eLogger(e *Engine) error {
	if e.Logger == nil {
		e.Logger = log.New(os.Stdout, log.LInfo, log.DefaultNullFormatter())
	}
	return nil
}

func SetLogger(l log.Logger) Config {
	return NewConfig(102,
		func(e *Engine) error {
			//e.SwapFormatter(log.GetFormatter(k))
			e.Logger = l
			return nil
		})
}

func eError(e *Engine) error {
	e.ErrorHandler.Init(e)
	return nil
}

func eWorld(e *Engine) error {
	hefn := func(err error) {
		e.hefn(e, err)
	}
	world := core.NewWorld(hefn)
	e.World = world
	return nil
}

func eInner(e *Engine) error {
	if e.inner == nil {
		var ifn Inner
		switch {
		case e.debug:
			ifn = debugInner(e, e.World)
		default:
			ifn = defaultInner(e, e.World)
		}
		return setInner(ifn, e)
	}

	return nil
}

func SetInner(ifn Inner) Config {
	return NewConfig(500,
		func(e *Engine) error {
			return setInner(ifn, e)
		})
}

func setInner(i Inner, e *Engine) error {
	e.inner = i
	return nil
}
