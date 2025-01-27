package tcp

import (
	"github.com/saniyar-dev/xk6-tcp/tcp"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/tcp", new(tcp.RootModule))
}
