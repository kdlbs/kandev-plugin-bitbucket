// Command kandev-plugin-bitbucket is the backend half of this Kandev plugin.
// Kandev spawns it as a gRPC subprocess; pluginsdk.Serve owns the transport.
package main

import (
	"kandev-plugin-bitbucket/internal/plugin"

	"github.com/kdlbs/kandev/pluginsdk"
)

func main() {
	pluginsdk.Serve(plugin.NewRuntime())
}
