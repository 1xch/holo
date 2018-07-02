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
	debug     bool
	formatter string
	log.Logger
}

func newOptions() *Options {
	return &Options{
		false,
		"null",
		log.New(os.Stdout, log.LInfo, log.DefaultNullFormatter()),
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
	return c, flip.ExitNo
}

func tFlags(fs *flip.FlagSet, o *Options) *flip.FlagSet {
	fs.BoolVar(&o.debug, "debug", o.debug, "Run any actions in debug mode where available.")
	fs.StringVar(&o.formatter, "formatter", o.formatter, "Specify the log formatter. [null|raw|stdout]")
	return fs
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

var rExecuting = []execution{
	engineInit,
	engineRun,
}

func engineInit(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	o.Print("initialize engine")
	E, engineInitError = engine.New(
		engine.SetDebug(O.debug),
		engine.SetLogger(O.Logger),
	)
	if engineInitError != nil {
		O.Print(engineInitError)
		return c, flip.ExitFailure
	}
	o.Print("engine initialized")
	return c, flip.ExitNo
}

func engineRun(o *Options, c context.Context) (context.Context, flip.ExitStatus) {
	o.Print("running...")
	//spew.Dump(E)
	run := func() {
		go func() {
			E.Run()
		}()
	EV:
		for {
			select {
			case <-E.Pre:
				<-E.Post
			case <-E.ChKill:
				E.Close()
				break EV
			}
		}
	}
	run()
	return c, flip.ExitSuccess
}

func rFlags(fs *flip.FlagSet, o *Options) *flip.FlagSet {
	return fs
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
		SetGroup("top", -1, TopCommand()).
		SetGroup("run", 1, RunCommand())
}

func main() {
	c := context.Background()
	os.Exit(F.Execute(c, os.Args))
}
