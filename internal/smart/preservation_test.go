package smart

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestObjectRequiresOneBoundedJSONDocument(t *testing.T) {
	for _, raw := range []string{"{} {}", "{} garbage", "null", "[]"} {
		if _, e := object([]byte(raw)); e == nil {
			t.Fatal("accepted trailing or wrong root", raw)
		}
	}
	if _, e := object([]byte(" \n {} \n ")); e != nil {
		t.Fatal(e)
	}
}
func TestEffectiveDownloadAliasesAndExtraPrecedence(t *testing.T) {
	download := M{"address": "download.test"}
	ob := M{"protocol": "vless", "settings": M{"address": "192.0.2.1"}, "streamSettings": M{"splithttpSettings": M{"downloadSettings": download}}}
	if independentIP("p", map[string]M{"p": ob}, map[string]bool{}) {
		t.Fatal("domain download treated as bootstrap IP")
	}
	stream := ob["streamSettings"].(M)
	stream["xhttpSettings"] = M{"downloadSettings": M{"address": "ignored.test"}, "extra": M{"downloadSettings": M{"address": "192.0.2.2"}}}
	if got := endpointAddresses(ob); !reflect.DeepEqual(got, []string{"192.0.2.1", "192.0.2.2"}) {
		t.Fatal("wrong core precedence", got)
	}
	if !independentIP("p", map[string]M{"p": ob}, map[string]bool{}) {
		t.Fatal("shadowed download rejected")
	}
}
func TestExplicitSocketSemanticsSurviveSmart(t *testing.T) {
	socket := M{"domainStrategy": "ForceIPv6", "tcpKeepAliveIdle": json.Number("0"), "tcpKeepAliveInterval": json.Number("90")}
	before, _ := json.Marshal(socket)
	nested := M{"sockopt": socket}
	ob := M{"protocol": "vless", "streamSettings": M{"sockopt": socket, "xhttpSettings": M{"extra": M{"downloadSettings": nested}}}}
	applySmartSocketPolicy(ob)
	after, _ := json.Marshal(socket)
	if string(before) != string(after) {
		t.Fatal("explicit transport policy overwritten")
	}
}
