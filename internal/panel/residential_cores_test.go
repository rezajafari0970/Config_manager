package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"testing"
)

func TestResidentialCompatibilityReportPins(t *testing.T) {
	r := residential.Report{Version: residential.Version, TemplateVersion: convert.Version, Worker: "worker", InfrastructureEpoch: "epoch", CoreSHA256: "b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006", CoreID: "xray-26.5.9"}
	if !validateResidentialReport(r) {
		t.Fatal("official compatible report rejected")
	}
	r.CoreID = "xray-26.11.9"
	if validateResidentialReport(r) {
		t.Fatal("mismatched actual version accepted")
	}
	r.CoreID = "xray-26.5.9"
	r.Compatibility = []residential.CoreAttempt{{CoreID: "xray-26.5.9", CoreSHA256: r.CoreSHA256}}
	if !validateResidentialReport(r) {
		t.Fatal("compatible audit rejected")
	}
	r.Compatibility = append(r.Compatibility, r.Compatibility[0])
	if validateResidentialReport(r) {
		t.Fatal("duplicate core vote accepted")
	}
	r.Compatibility = nil
	r.CoreSHA256 = "untrusted"
	if validateResidentialReport(r) {
		t.Fatal("untrusted executable accepted")
	}
}
