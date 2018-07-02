package engine

import (
	"os"
	"time"

	"github.com/Laughs-In-Flowers/holo/lib/core"
	"github.com/Laughs-In-Flowers/log"
)

type Settings struct{}

func (s *Settings) resetSettings() {}

type State struct {
	debug   bool
	restart bool
	kill    bool
	ChKill  chan struct{}
	Pre     chan struct{}
	Post    chan struct{}
	ChSys   chan os.Signal
}

func newState() *State {
	return &State{
		false,
		false,
		false,
		make(chan struct{}, 0),
		make(chan struct{}, 0),
		make(chan struct{}, 0),
		make(chan os.Signal, 0),
	}
}

func (s *State) resetState() {
	s.restart = false
	s.kill = false
	s.debug = false
}

func (s *State) Restart() {
	s.restart = true
}

func (s *State) Kill() {
	s.kill = true
}

func (s *State) Debug() bool {
	return s.debug
}

type components struct {
	inner Inner
	World core.World
}

type Engine struct {
	log.Logger
	ErrorHandler
	Configuration
	Settings
	*State
	components
}

func New(cnf ...Config) (*Engine, error) {
	e := new(Engine)
	e.Configuration.Init(e, cnf...)
	err := e.Configure()
	if err != nil {
		return nil, err
	}
	return e, nil
}

type Inner func()

func defaultInner(e *Engine, w core.World) Inner {
	return func() {
		for {
			switch {
			case e.kill:
				e.ChKill <- struct{}{}
			default:
				e.Pre <- struct{}{}
				w.Update(time.Now().Unix())
				e.Post <- struct{}{}
			}
		}
	}
}

func debugInner(e *Engine, w core.World) Inner {
	return func() {
	RESTART:
		f := DebugFrame()
		for {
			switch {
			case e.restart:
				e.restart = false
				e.Print("restarting...")
				goto RESTART
			case e.kill:
				e.ChKill <- struct{}{}
			default:
				f.Start()
				w.Update(time.Now().Unix())
				f.End()
			}
		}
	}
}

func (e *Engine) Run() {
	inr := e.inner
	e.Print("running...")
	inr()
}

func (e *Engine) Close() {
	switch {
	case e.last != nil:
		e.Print("closing with error")
		e.Print(e.last)
	default:
		e.Print("closing....")
	}
	e.Print("done")
	os.Exit(0)
}
