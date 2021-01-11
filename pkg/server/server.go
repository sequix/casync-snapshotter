package server

import (
	"context"
	"flag"
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sequix/casync-snapshotter/pkg/buildinfo"
	"github.com/sequix/casync-snapshotter/pkg/log"
	"github.com/sequix/casync-snapshotter/pkg/util"
)

var (
	flagAddr       = flag.String("svr-addr", ":8996", "address http server listens to")
	flagMetricsOff = flag.Bool("svr-metrics-off", false, "disable /metrics handler")
	flagPprofOn    = flag.Bool("svr-pprof-on", false, "enable /debug/pprof handler")
)

var (
	mtxDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "bec_alert_http_duration",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"path", "method", "code"})

	mtxReqSize = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "bec_alert_http_request_bytes",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"path", "method", "code"})

	mtxResSize = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name:       "bec_alert_http_response_bytes",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"path", "method", "code"})
)

var (
	svr *http.Server
	mux *http.ServeMux
)

func Init() {
	mux = http.NewServeMux()
	svr = &http.Server{
		Addr:    *flagAddr,
		Handler: mux,
	}

	if !*flagMetricsOff {
		flag.VisitAll(func(f *flag.Flag) {
			value := f.Value.String()
			if isSecureArg(f.Name) {
				value = "secret"
			}
			c := promauto.NewCounter(prometheus.CounterOpts{
				Name: "flag",
				ConstLabels: map[string]string{
					"name":  f.Name,
					"value": value,
				},
			})
			c.Inc()
		})
		c := promauto.NewCounter(prometheus.CounterOpts{
			Name: "commit",
			ConstLabels: map[string]string{
				"id": buildinfo.Commit,
			},
		})
		c.Inc()
		Register("/metrics", promhttp.Handler())
	}

	if *flagPprofOn {
		Register("/debug/pprof/alloca", pprof.Handler("allocs"))
		Register("/debug/pprof/block", pprof.Handler("block"))
		Register("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		Register("/debug/pprof/goroutine", pprof.Handler("goroutine"))
		Register("/debug/pprof/heap", pprof.Handler("heap"))
		Register("/debug/pprof/mutex", pprof.Handler("mutex"))
		Register("/debug/pprof/profile", pprof.Handler("profile"))
		Register("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
		Register("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	}

	Register("health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func isSecureArg(name string) bool {
	return strings.Contains(name, "password") || strings.Contains(name, "sk")
}

func Run(stop util.BroadcastCh) {
	go func() {
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.G.WithError(err).Error("http server ListenAndServe")
		}
	}()
	stop.Wait()
	if err := svr.Shutdown(context.Background()); err != nil {
		log.G.WithError(err).Error("http server shutdown")
	}
}

func Register(path string, f http.Handler) {
	if *flagMetricsOff {
		registerWithoutMetrics(path, f)
	}
	registerWithMetrics(path, f)
}

func registerWithoutMetrics(path string, f http.Handler) {
	mux.Handle(path, f)
}

func registerWithMetrics(path string, f http.Handler) {
	cm := map[string]string{"path": path}
	f1 := promhttp.InstrumentHandlerDuration(mtxDuration.MustCurryWith(cm), f)
	f2 := promhttp.InstrumentHandlerRequestSize(mtxReqSize.MustCurryWith(cm), f1)
	f3 := promhttp.InstrumentHandlerResponseSize(mtxResSize.MustCurryWith(cm), f2)
	mux.Handle(path, f3)
}