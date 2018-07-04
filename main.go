package main

import (
	"context"
	"os"
	"path"

	"github.com/Laughs-In-Flowers/flip"
	"github.com/Laughs-In-Flowers/holo/lib/engine"
	"github.com/Laughs-In-Flowers/log"
)

type Options struct {
	log.Logger
	*tOptions
	*dOptions
	*rOptions
}

func newOptions() *Options {
	return &Options{
		log.New(os.Stdout, log.LInfo, log.DefaultNullFormatter()),
		defaultTOptions(),
		defaultDOptions(),
		defaultROptions(),
	}
}

func cExecute(o *Options, c context.Context, a []string, x ...execution) (context.Context, flip.ExitStatus) {
	var status flip.ExitStatus
	for _, fn := range x {
		c, status = fn(o, c)
		if status != flip.ExitNo {
			return c, status
		}
	}
	return c, flip.ExitNo
}

type execution func(o *Options, c context.Context) (context.Context, flip.ExitStatus)

var tExecuting = []execution{
	logSetting,
}

func logSetting(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	if o.formatter != "null" {
		switch o.formatter {
		case "text", "stdout":
			o.SwapFormatter(log.GetFormatter("holo_text"))
		default:
			o.SwapFormatter(log.GetFormatter(o.formatter))
		}
	}

	eiz = append(eiz, engine.SetLogger(O.Logger))

	return c, flip.ExitNo
}

func tFlags(fs *flip.FlagSet, o *Options) *flip.FlagSet {
	fs.StringVar(&o.formatter, "formatter", o.formatter, "Specify the log formatter. [null|raw|stdout]")
	return fs
}

type tOptions struct {
	formatter string
}

func defaultTOptions() *tOptions {
	return &tOptions{"null"}
}

func TopCommand() flip.Command {
	fs := flip.NewFlagSet("top", flip.ContinueOnError)
	fs = tFlags(fs, O)

	return flip.NewCommand(
		"",
		"holo",
		"Top level options use.",
		1,
		false,
		func(c context.Context, a []string) (context.Context, flip.ExitStatus) {
			return cExecute(O, c, a, tExecuting...)
		},
		fs,
	)
}

var dExecuting = []execution{
	debugSetting,
}

func debugSetting(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	o.debug = true

	eiz = append(eiz,
		engine.SetDebug(o.debug),
		engine.SetReportStep(o.reportStep),
		engine.SetReportFrame(o.reportFrame),
	)

	return c, flip.ExitNo
}

func dFlags(fs *flip.FlagSet, o *Options) *flip.FlagSet {
	fs.BoolVar(&o.reportStep, "reportStep", o.reportStep, "log step information")
	fs.BoolVar(&o.reportFrame, "reportFrame", o.reportFrame, "log frame information")
	return fs
}

type dOptions struct {
	debug       bool
	reportStep  bool
	reportFrame bool
}

func defaultDOptions() *dOptions {
	return &dOptions{false, false, false}
}

func DebugCommand() flip.Command {
	fs := flip.NewFlagSet("debug", flip.ContinueOnError)
	fs = dFlags(fs, O)

	return flip.NewCommand(
		"",
		"debug",
		"Debug options use. Any flags here will set Engine debug to true",
		1,
		false,
		func(c context.Context, a []string) (context.Context, flip.ExitStatus) {
			return cExecute(O, c, a, dExecuting...)
		},
		fs,
	)
}

var (
	rExecuting = []execution{
		engineInit,
		engineRun,
	}

	eiz = []engine.Config{}
)

func engineInit(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	o.Print("initialize engine")

	var inr engine.MakeInner = engine.DefaultInner

	switch {
	case O.debug:
		inr = engine.DebugInner
	case O.noTickDuration:
		inr = engine.NoDurationLimitInner
	}

	eiz = append(eiz,
		engine.SetInner(inr),
		engine.SetTickDuration(O.tickDuration),
		engine.SetTickValue(O.tickValue),
		engine.SetLastTick(O.lastTick),
	)

	E, engineInitError = engine.New(eiz...)

	if engineInitError != nil {
		O.Print(engineInitError)
		return c, flip.ExitFailure
	}
	o.Print("engine initialized")
	return c, flip.ExitNo
}

func retSignal(out int) flip.ExitStatus {
	switch out {
	case 0:
		return flip.ExitSuccess
	case -1:
		return flip.ExitFailure
	}
	return flip.ExitUsageError
}

func engineRun(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	var ret int = -2
	run := func() {
		go func() {
			E.Run()
		}()
	EV:
		for {
			select {
			case sig := <-E.ChSys:
				ret = E.SignalHandler(sig)
				break EV
			case <-E.ChKill:
				ret = E.Close()
				break EV
			}
		}
	}
	run()
	return c, retSignal(ret)
}

func rFlags(fs *flip.FlagSet, o *Options) *flip.FlagSet {
	fs.BoolVar(&o.noTickDuration, "noTickDuration", o.noTickDuration, "Ignore tick duration in inner loop, does not override debug(which sets its own inner loop).")
	fs.StringVar(&o.tickDuration, "tickDuration", o.tickDuration, "The duration between world processing steps.")
	fs.Float64Var(&o.tickValue, "tickValue", o.tickValue, "The tick value to increment by on world processing steps.")
	fs.Float64Var(&o.lastTick, "lastTick", o.lastTick, "Stop engine running when this tick value is reached.")
	return fs
}

type rOptions struct {
	noTickDuration      bool
	tickDuration        string
	tickValue, lastTick float64
}

func defaultROptions() *rOptions {
	return &rOptions{false, "1ns", 1.0, 0.0}
}

func RunCommand() flip.Command {
	fs := flip.NewFlagSet("run", flip.ContinueOnError)
	fs = rFlags(fs, O)

	return flip.NewCommand(
		"",
		"run",
		"run a holo instance.",
		1,
		false,
		func(c context.Context, a []string) (context.Context, flip.ExitStatus) {
			return cExecute(O, c, a, rExecuting...)
		},
		fs,
	)
}

var (
	E               *engine.Engine
	engineInitError error
	O               *Options
	F               flip.Flip
	versionPackage  string = path.Base(os.Args[0])
	versionTag      string = "no tag"
	versionHash     string = "no hash"
	versionDate     string = "no date"
)

func init() {
	log.SetFormatter("holo_text", log.MakeTextFormatter(versionPackage))
	O = newOptions()
	F = flip.New("holo")
	F.AddCommand("version", versionPackage, versionTag, versionHash, versionDate).
		AddCommand("help").
		SetGroup("top", -1, TopCommand(), DebugCommand()).
		SetGroup("run", 1, RunCommand())
}

func main() {
	c := context.Background()
	os.Exit(F.Execute(c, os.Args))
}
