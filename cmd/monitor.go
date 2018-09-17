package cmd

import (
	"expvar"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

type monitor struct {
	info    *expvar.Map
	stopped chan struct{}
	server  *http.Server

	BurnIn         *expvar.Int
	ConvergeWindow *expvar.Int
	BaseChains     *expvar.Int
	TotalChains    *expvar.Int
	MaxIters       *expvar.Int
	MaxSeconds     *expvar.Int
	RunTime        *expvar.Float
	TotalSamples   *expvar.Int
	Iterations     *expvar.Int

	LastMeanHellinger *expvar.Float
	LastMaxHellinger  *expvar.Float
	LastMeanJSD       *expvar.Float
	LastMaxJSD        *expvar.Float
}

// Start begins the monitor
func (m *monitor) Start() error {
	if m.info != nil {
		return errors.Errorf("BUG: You may only start the process monitor once")
	}

	m.info = expvar.NewMap("grample-progress")
	m.stopped = make(chan struct{})
	m.server = &http.Server{
		Addr: ":8000", // TODO: allow override in call to start
	}

	// Help the user and redirect to the only thing currently available:
	// the handler from the expvar package
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/debug/vars", http.StatusTemporaryRedirect)
	})

	m.BurnIn = expvar.NewInt("Burn-In")
	m.ConvergeWindow = expvar.NewInt("Convergence-Window")
	m.BaseChains = expvar.NewInt("Base-Chain-Count")
	m.TotalChains = expvar.NewInt("Total-Chain-Count")
	m.MaxIters = expvar.NewInt("Max-Iterations")
	m.MaxSeconds = expvar.NewInt("Max-Seconds")
	m.RunTime = expvar.NewFloat("Run-Time")
	m.TotalSamples = expvar.NewInt("Total-Samples")
	m.Iterations = expvar.NewInt("Iterations")

	m.LastMeanHellinger = expvar.NewFloat("Last-Mean-Hellinger")
	m.LastMaxHellinger = expvar.NewFloat("Last-Max-Hellinger")
	m.LastMeanJSD = expvar.NewFloat("Last-Mean-JSD")
	m.LastMaxJSD = expvar.NewFloat("Last-Max-JSD")

	// Actual server that will close the stopped channel on exit
	started := make(chan struct{})
	go func() {
		defer close(m.stopped)
		fmt.Fprintf(os.Stderr, "HTTP now available at %v (see debug/vars/)\n", m.server.Addr)
		close(started)
		m.server.ListenAndServe()
	}()

	<-started
	return nil
}

func (m *monitor) Stop() {
	if m.info == nil {
		return
	}

	m.server.Close()

	select {
	case <-m.stopped:
		fmt.Fprintf(os.Stderr, "HTTP Info Stopped\n")
	case <-time.After(2 * time.Second):
		fmt.Fprintf(os.Stderr, "HTTP would NOT stop: just continuing on\n")
	}
}
