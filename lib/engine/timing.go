package engine

import (
	"time"
)

type FrameFunc func(*frame)

type FPSFunc func(*frame, time.Duration) (float64, float64, bool)

// Frame is an interface for managing and detailing engine timing information.
type Frame interface {
	Start()
	End()
	FPS(time.Duration) (float64, float64, bool)
}

type frame struct {
	start          FrameFunc     //
	end            FrameFunc     //
	fps            FPSFunc       //
	targetFPS      uint          // desired number of frames per second
	targetDuration time.Duration // calculated desired duration of frame
	frameStart     time.Time     // start time of last frame
	frameTimes     time.Duration // accumulated frame times for potential FPS calculation
	frameCount     uint          // accumulated number of frames for FPS calculation
	lastUpdate     time.Time     // time of last FPS calculation update
	timer          *time.Timer   // timer for sleeping during frame
}

func newFrame(start, end FrameFunc, fps FPSFunc, conf ...FrameFunc) *frame {
	switch {
	case start == nil, end == nil, fps == nil:
		panic("frame & fps functions cannot be nil")
	}
	f := &frame{
		start:      start,
		end:        end,
		fps:        fps,
		lastUpdate: time.Now(),
		timer:      time.NewTimer(0),
	}

	for _, c := range conf {
		c(f)
	}

	<-f.timer.C
	return f
}

func defaultFPS(f *frame, t time.Duration) (float64, float64, bool) {
	elapsed := time.Now().Sub(f.lastUpdate)
	if elapsed < t {
		return 0, 0, false
	}
	fps := float64(f.frameCount) / elapsed.Seconds()
	frameDur := f.frameTimes.Seconds() / float64(f.frameCount)
	pfps := 1.0 / frameDur
	f.frameCount = 0
	f.frameTimes = 0
	f.lastUpdate = time.Now()
	return fps, pfps, true
}

func elapse(f *frame) time.Duration {
	elapsed := time.Now().Sub(f.frameStart)
	f.frameCount++
	f.frameTimes += elapsed
	return elapsed
}

func DebugFrame() Frame {
	return newFrame(
		func(f *frame) { f.frameStart = time.Now() },
		func(f *frame) { elapse(f) },
		defaultFPS,
	)
}

func MaxLimitFrame(target uint) Frame {
	return newFrame(
		func(f *frame) { f.frameStart = time.Now() },
		func(f *frame) {
			elapsed := elapse(f)
			diff := f.targetDuration - elapsed
			if diff > 0 {
				f.timer.Reset(diff)
				<-f.timer.C
			}
		},
		defaultFPS,
		func(f *frame) {
			if target < 1 {
				target = 60
			}
			f.targetFPS = target
			f.targetDuration = time.Second / time.Duration(target)
		},
	)
}

func (f *frame) Start() {
	f.start(f)
}

func (f *frame) End() {
	f.end(f)
}

func (f *frame) FPS(t time.Duration) (float64, float64, bool) {
	return f.fps(f, t)
}

func frameReport(e *Engine, f Frame) {
	if fps, pfps, b := f.FPS(1 * time.Second); b {
		e.Printf("fps: %f / pfps: %f", fps, pfps)
	}
}
