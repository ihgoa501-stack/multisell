package operationlog

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var auditWritesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "multisell_audit_writes_total",
	Help: "Audit writes performed through the operation log service by result.",
}, []string{"result"})

var auditIntegrity = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "multisell_audit_integrity",
	Help: "Audit hash-chain integrity: 1 valid, 0 broken/check failed, -1 not checked yet.",
})

func init() { auditIntegrity.Set(-1) }
