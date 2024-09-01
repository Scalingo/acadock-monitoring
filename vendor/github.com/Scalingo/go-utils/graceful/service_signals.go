package graceful

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Scalingo/go-utils/logger"
)

// setupSignals is catching INT/TERM signals to handle a graceful shutdown operation
// and HUP for a graceful restart. In the case of a restart, the socket is given to the
// child process to keep receiving new connections while waiting for the old one to finish
// properly.
func (s *Service) setupSignals(ctx context.Context) {
	log := logger.Get(ctx)
	ch := make(chan os.Signal, 10)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	for {
		sig := <-ch
		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			s.upg.Stop()
			return
		case syscall.SIGHUP:
			log.Info("Request graceful restart")
			err := s.upg.Upgrade()
			if err != nil {
				log.WithError(err).Error("Fail to start new service")
			}
		}
	}
}
