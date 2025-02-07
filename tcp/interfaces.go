// Package tcp implements ways to test a tcp connection on k6 and extends the ability of k6 by extension name xk6-tcp
package tcp

import (
	"github.com/grafana/sobek"
	"go.k6.io/k6/js/modules"
)

// ExportedAPI interface is an interface which all exported api needs to implement this interface
type ExportedAPI interface {
	modules.Instance
	init(sobek.ConstructorCall) *sobek.Object
}
