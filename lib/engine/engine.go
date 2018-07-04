package engine

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Laughs-In-Flowers/holo/lib/core"
	"github.com/Laughs-In-Flowers/holo/lib/util/step"
	"github.com/Laughs-In-Flowers/holo/lib/util/xrr"
	"github.com/Laughs-In-Flowers/log"
)

//
type Settings struct {
	HardExit                    bool
	TickDuration                time.Duration
	TickIncr, TickInit, TickEnd float64
	DebugReportStep             bool
	DebugReportFrame            bool
}

func (s *Settings) resetSettings() {
	s.HardExit = false
	s.TickDuration = 1 * time.Nanosecond
	s.TickIncr = 1.0
	s.TickInit = 0.0
}

//
type Close func(*Engine)

var defaultClose = []Close{
	func(e *Engine) { e.Printf("last tick: %f", e.LastTick) },
}

//
type State struct {
	debug    bool
	lock     bool
	restart  bool
	kill     bool
	ChKill   chan struct{}
	ChSys    chan os.Signal
	close    []Close
	LastTick float64
}

func newState() *State {
	s := &State{
		false,
		false,
		false,
		false,
		make(chan struct{}, 0),
		make(chan os.Signal, 0),
		defaultClose,
		0,
	}

	signal.Notify(
		s.ChSys,
	)

	return s
}

func (s *State) resetState() {
	s.restart = false
	s.kill = false
	s.debug = false
}

//
func (s *State) Pause() {
	s.lock = true
}

//
func (s *State) Unpause() {
	s.lock = false
}

//
func (s *State) Restart() {
	s.restart = true
}

//
func (s *State) Kill() {
	s.kill = true
}

//
func (s *State) Debug() bool {
	return s.debug
}

//
func (s *State) SetClose(c ...Close) {
	s.close = append(s.close, c...)
}

func (s *State) execClose(e *Engine) {
	for _, v := range s.close {
		v(e)
	}
}

type components struct {
	inner Inner
	World core.World
}

//
type Engine struct {
	log.Logger
	ErrorHandler
	Configuration
	Settings
	*State
	components
}

//
func New(cnf ...Config) (*Engine, error) {
	e := new(Engine)
	e.Configuration.Init(e, cnf...)
	err := e.Configure()
	if err != nil {
		return nil, err
	}
	return e, nil
}

//
type Inner func()

func NoDurationLimitInner(e *Engine, w core.World) Inner {
	s := step.New(e.TickDuration, e.TickInit)
	return func() {
		for {
			s.Increment(e.TickIncr)
			switch {
			case e.lock:
				// do nothing
			case e.kill:
				e.ChKill <- struct{}{}
			default:
				w.Update(s)
			}
			killIf(e, s)
		}
	}
}

func DefaultInner(e *Engine, w core.World) Inner {
	s := step.New(e.TickDuration, e.TickInit)
	return func() {
		for range s.C {
			s.Increment(e.TickIncr)
			switch {
			case e.lock:
				// do nothing
			case e.kill:
				e.ChKill <- struct{}{}
			default:
				w.Update(s)
			}
			killIf(e, s)
		}
	}
}

func DebugInner(e *Engine, w core.World) Inner {
	return func() {
	RESTART:
		s := step.New(e.TickDuration, e.TickInit)
		f := DebugFrame()
		for range s.C {
			s.Increment(e.TickIncr)
			switch {
			case e.lock:
				// do nothing
			case e.restart:
				e.restart = false
				e.Print("restarting...")
				goto RESTART
			case e.kill:
				e.ChKill <- struct{}{}
			default:
				f.Start()
				w.Update(s)
				f.End()
				e.DebugReport(f, s)
			}
			killIf(e, s)
		}
	}
}

func (e *Engine) DebugReport(f Frame, s *step.Step) {
	if e.DebugReportStep {
		e.Printf("step: %f", s.Value)
	}
	if e.DebugReportFrame {
		frameReport(e, f)
	}
}

func killIf(e *Engine, s *step.Step) {
	e.LastTick = s.Value
	if e.TickEnd != 0.0 && s.Value == e.TickEnd {
		e.lock = true
		e.ChKill <- struct{}{}
	}
}

//
func (e *Engine) Run() {
	inr := e.inner
	e.Print("running...")
	inr()
}

var forcedSignalError = xrr.Xrror("signal[%v] forcing immediate shutdown").Out

// Handle the provided os.Signal, clsoing if necessary and returning an integer.
func (e *Engine) SignalHandler(s os.Signal) int {
	e.Printf("got signal: %v", s)
	switch s {
	case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
		return e.Close()
	case syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP,
		syscall.SIGABRT, syscall.SIGSTKFLT /*,syscall.SIGEMT*/, syscall.SIGSYS:
		e.last = forcedSignalError(s)
		return e.Close()
	}
	return -1
}

// Handles closing, returns an exit code only unless settings.HardExit is true
func (e *Engine) Close() int {
	e.execClose(e)
	var ret int = 0
	switch {
	case e.last != nil:
		e.Print("closing with error")
		e.Print(e.last)
		ret = -1
	default:
		e.Print("closing...")
	}
	e.Print("done")
	if e.HardExit {
		os.Exit(ret)
	}
	return ret
}
