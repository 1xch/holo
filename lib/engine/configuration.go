package engine

import (
	"fmt"
	"os"
	"sort"
	"time"

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

type confReport struct {
	h []string
}

func (c *confReport) Add(s string) {
	c.h = append(c.h, s)
}

var r *confReport

var builtIns = []Config{
	config{-1, eReportInit},
	config{001, eState},
	config{002, eDefaults},
	config{101, eLogger},
	config{102, eError},
	config{501, eWorld},
	config{601, eInner},
	config{999, eReportEnd},
}

func eReportInit(e *Engine) error {
	r = new(confReport)
	return nil
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
			r.Add(fmt.Sprintf("debug is %t", e.debug))
			return nil
		})
}

func SetReportStep(b bool) Config {
	return NewConfig(50,
		func(e *Engine) error {
			e.DebugReportStep = b
			r.Add(fmt.Sprintf("debug reportStep is %t", e.DebugReportStep))
			return nil
		})
}

func SetReportFrame(b bool) Config {
	return NewConfig(50,
		func(e *Engine) error {
			e.DebugReportFrame = b
			r.Add(fmt.Sprintf("debug reportFrame is %t", e.DebugReportFrame))
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

type MakeInner func(e *Engine, w core.World) Inner

func eInner(e *Engine) error {
	if e.inner == nil {
		var ifn MakeInner = DefaultInner
		if e.debug {
			ifn = DebugInner
		}
		return setInner(ifn, e)
	}

	return nil
}

func SetInner(fn MakeInner) Config {
	return NewConfig(600,
		func(e *Engine) error {
			return setInner(fn, e)
		})
}

func setInner(fn MakeInner, e *Engine) error {
	e.inner = fn(e, e.World)
	return nil
}

func SetTickDuration(d string) Config {
	return NewConfig(500,
		func(e *Engine) error {
			dur, err := time.ParseDuration(d)
			if err != nil {
				return err
			}
			e.TickDuration = dur
			return nil
		})
}

func SetTickValue(v float64) Config {
	return NewConfig(500,
		func(e *Engine) error {
			e.TickIncr = v
			return nil
		})
}

func SetLastTick(v float64) Config {
	return NewConfig(500,
		func(e *Engine) error {
			e.TickEnd = v
			return nil
		})
}

func eReportEnd(e *Engine) error {
	if len(r.h) > 0 {
		for _, rc := range r.h {
			e.Print(rc)
		}
	}
	return nil
}
