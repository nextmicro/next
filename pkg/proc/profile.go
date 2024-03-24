//go:build linux || darwin

package proc

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nextmicro/logger"
)

// DefaultMemProfileRate is the default memory profiling rate.
// See also http://golang.org/pkg/runtime/#pkg-variables
const DefaultMemProfileRate = 4096

// started is non zero if a profile is running.
var started uint32

// Profile represents an active profiling session.
type Profile struct {
	// closers holds cleanup functions that run after each profile
	closers []func()

	// stopped records if a call to profile.Stop has been made
	stopped uint32
}

func (p *Profile) close() {
	for _, closer := range p.closers {
		closer()
	}
}

func (p *Profile) startBlockProfile() {
	fn := createDumpFile("block")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create block profile %q: %v", fn, err)
		return
	}

	runtime.SetBlockProfileRate(1)
	logger.Infof("profile: block profiling enabled, %s", fn)
	p.closers = append(p.closers, func() {
		if err = pprof.Lookup("block").WriteTo(f, 0); err != nil {
			logger.Errorf("profile: could not write block profile: %v", err)
		}
		if err = f.Close(); err != nil {
			logger.Errorf("profile: could not close block profile file: %v", err)
		}
		runtime.SetBlockProfileRate(0)
		logger.Infof("profile: block profiling disabled, %s", fn)
	})
}

func (p *Profile) startCpuProfile() {
	fn := createDumpFile("cpu")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create cpu profile %q: %v", fn, err)
		return
	}

	logger.Infof("profile: cpu profiling enabled, %s", fn)
	if err = pprof.StartCPUProfile(f); err != nil {
		logger.Errorf("profile: could not start cpu profile: %v", err)
		return
	}
	p.closers = append(p.closers, func() {
		pprof.StopCPUProfile()
		if err = f.Close(); err != nil {
			logger.Errorf("profile: could not close cpu profile file: %v", err)
		}
		logger.Infof("profile: cpu profiling disabled, %s", fn)
	})
}

func (p *Profile) startMemProfile() {
	fn := createDumpFile("mem")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create memory profile %q: %v", fn, err)
		return
	}

	old := runtime.MemProfileRate
	runtime.MemProfileRate = DefaultMemProfileRate
	logger.Infof("profile: memory profiling enabled (rate %d), %s", runtime.MemProfileRate, fn)
	p.closers = append(p.closers, func() {
		if err = pprof.Lookup("heap").WriteTo(f, 0); err != nil {
			logger.Errorf("profile: could not write memory profile: %v", err)
		}
		if err = f.Close(); err != nil {
			logger.Errorf("profile: could not close memory profile file: %v", err)
		}
		runtime.MemProfileRate = old
		logger.Infof("profile: memory profiling disabled, %s", fn)
	})
}

func (p *Profile) startMutexProfile() {
	fn := createDumpFile("mutex")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create mutex profile %q: %v", fn, err)
		return
	}

	runtime.SetMutexProfileFraction(1)
	logger.Infof("profile: mutex profiling enabled, %s", fn)
	p.closers = append(p.closers, func() {
		if mp := pprof.Lookup("mutex"); mp != nil {
			err = mp.WriteTo(f, 0)
		}
		err = f.Close()
		if err != nil {
			logger.Errorf("profile: could not close mutex profile file: %v", err)
		}
		runtime.SetMutexProfileFraction(0)
		logger.Infof("profile: mutex profiling disabled, %s", fn)
	})
}

func (p *Profile) startThreadCreateProfile() {
	fn := createDumpFile("threadcreate")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create threadcreate profile %q: %v", fn, err)
		return
	}

	logger.Infof("profile: threadcreate profiling enabled, %s", fn)
	p.closers = append(p.closers, func() {
		if mp := pprof.Lookup("threadcreate"); mp != nil {
			err = mp.WriteTo(f, 0)
		}
		err = f.Close()
		if err != nil {
			logger.Errorf("profile: could not close threadcreate profile file: %v", err)
		}
		logger.Infof("profile: threadcreate profiling disabled, %s", fn)
	})
}

func (p *Profile) startTraceProfile() {
	fn := createDumpFile("trace")
	f, err := os.Create(fn)
	if err != nil {
		logger.Errorf("profile: could not create trace output file %q: %v", fn, err)
		return
	}

	if err := trace.Start(f); err != nil {
		logger.Errorf("profile: could not start trace: %v", err)
		return
	}

	logger.Infof("profile: trace enabled, %s", fn)
	p.closers = append(p.closers, func() {
		trace.Stop()
		logger.Infof("profile: trace disabled, %s", fn)
	})
}

// Stop stops the profile and flushes any unwritten data.
func (p *Profile) Stop() {
	if !atomic.CompareAndSwapUint32(&p.stopped, 0, 1) {
		// someone has already called close
		return
	}
	p.close()
	atomic.StoreUint32(&started, 0)
}

// StartProfile starts a new profiling session.
// The caller should call the Stop method on the value returned
// to cleanly stop profiling.
func StartProfile() Stopper {
	if !atomic.CompareAndSwapUint32(&started, 0, 1) {
		logger.Error("profile: Start() already called")
		return noopStopper
	}

	var prof Profile
	prof.startCpuProfile()
	prof.startMemProfile()
	prof.startMutexProfile()
	prof.startBlockProfile()
	prof.startTraceProfile()
	prof.startThreadCreateProfile()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		<-c

		logger.Info("profile: caught interrupt, stopping profiles")
		prof.Stop()

		signal.Reset()
		if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
			logger.Errorf("profile: failed to send interrupt signal: %v", err)
		}
	}()

	return &prof
}

func createDumpFile(kind string) string {
	command := path.Base(os.Args[0])
	pid := syscall.Getpid()
	return path.Join(os.TempDir(), fmt.Sprintf("%s-%d-%s-%s.pprof",
		command, pid, kind, time.Now().Format(timeFormat)))
}
