package tcp

import (
	"github.com/grafana/sobek"
	"go.k6.io/k6/js/modules"
)

type RootModule struct{}

var _ modules.Module = &RootModule{}

func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &TCPAPI{
		vu: vu,
	}
}

type TCPAPI struct {
	vu modules.VU
	// blobConstructor sobek.Value
}

var _ modules.Instance = &TCPAPI{}

// Exports implements the modules.Instance interface's Exports
func (r *TCPAPI) Exports() modules.Exports {
	// r.blobConstructor = r.vu.Runtime().ToValue(r.blob)
	return modules.Exports{
		Named: map[string]interface{}{
			"TCP": r.tcp,
			// "Blob": r.blobConstructor,
		},
	}
}

func (r *TCPAPI) tcp(c sobek.ConstructorCall) *sobek.Object {
	return &sobek.Object{}
}
