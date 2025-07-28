// internal/workload/workload.go
package workload

import (
	"github.com/elchinoo/stormdb/pkg/plugin"
)

// Workload is an alias for the plugin.Workload interface to maintain
// backward compatibility with existing code
type Workload = plugin.Workload
