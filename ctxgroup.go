package main

import (
  "context"
  "os"
  "os/signal"
  "syscall"
  "golang.org/x/sync/errgroup"
)

func buildErrGroup(ctx context.Context) SignalledErrGroup {
  cctx, done := context.WithCancel(ctx)

  g, gctx := errgroup.WithContext(cctx)

  g.Go(func() error {
    signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

		select {
		case <-signalChannel:
			done()
		case <-gctx.Done():
			return gctx.Err()
		}

		return nil
  })

  return &sigerrgroup{
    g: g,
    gctx: gctx,
  }

}

// SignalledErrGroup is a struct to handle signals and errors across goroutines
type SignalledErrGroup interface {
  Go(func(ctx context.Context) error)
  Wait() error
}

type sigerrgroup struct {
  g *errgroup.Group
  gctx context.Context
}

func (s *sigerrgroup) Go(fn func(ctx context.Context) error) {
  s.g.Go(func() error {
    res := fn(s.gctx)
    if res != nil && res != context.Canceled {
      return res
    }
    return nil
  })
}

func (s *sigerrgroup) Wait() error {
  return s.g.Wait()
}

