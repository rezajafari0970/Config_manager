// Run inside the existing closed sandbox, never with production credentials.
// Reproduces AndroidLibXrayLite 2ffe5f794a32 (Lib v40) MeasureOutboundDelay's exact
// essential-app filter against Xray a4a3dff5b20c. This is not an APK test.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	coreserial "github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	result := map[string]any{"passed": false, "core": core.Version(), "android_apk": "not_run"}
	if err := run(result); err != nil {
		result["error"] = err.Error()
	}
	_ = json.NewEncoder(os.Stdout).Encode(result)
}
func run(result map[string]any) error {
	b, err := os.ReadFile(os.Args[len(os.Args)-1])
	if err != nil {
		return err
	}
	config, err := coreserial.LoadJSONConfig(strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	config.Inbound = nil
	var essentialApp []*serial.TypedMessage
	for _, app := range config.App {
		if app.Type == "xray.app.proxyman.OutboundConfig" || app.Type == "xray.app.dispatcher.Config" || app.Type == "xray.app.log.Config" {
			essentialApp = append(essentialApp, app)
		}
	}
	config.App = essentialApp
	inst, err := core.New(config)
	if err != nil {
		return err
	}
	if err := inst.Start(); err != nil {
		return err
	}
	defer inst.Close()
	tr := &http.Transport{DisableKeepAlives: true, TLSHandshakeTimeout: 6 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dest, err := xnet.ParseDestination(network + ":" + addr)
			if err != nil {
				return nil, err
			}
			return core.Dial(ctx, inst, dest)
		}}
	defer tr.CloseIdleConnections()
	client := &http.Client{Transport: tr, Timeout: 3 * time.Second}
	start := time.Now()
	response, err := client.Get("http://127.0.0.1:24447/generate_204")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return fmt.Errorf("status %d", response.StatusCode)
	}
	result["passed"] = true
	result["elapsed_ms"] = time.Since(start).Milliseconds()
	return nil
}
