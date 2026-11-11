package compat

import "testing"

func TestSocketDNSFamiliesAndNestedDownload(t *testing.T) {
	for input, expected := range map[string]string{"UseIPv4": "ForceIPv4", "UseIP": "ForceIP", "AsIs": "ForceIP", "UseIPv6": "ForceIPv6", "UseIPv4v6": "ForceIPv4v6", "UseIPv6v4": "ForceIPv6v4", "ForceIPv6": "ForceIPv6"} {
		opt := M{"domainStrategy": input, "dialerProxy": "carrier"}
		download := M{"sockopt": opt, "security": "tls"}
		stream := M{"xhttpSettings": M{"extra": M{"downloadSettings": download}}}
		if err := FailClosedSocketDNS(stream); err != nil {
			t.Fatal(err)
		}
		if opt["domainStrategy"] != expected || opt["dialerProxy"] != "carrier" || download["security"] != "tls" {
			t.Fatalf("changed transport semantics: %#v", download)
		}
		if stream["sockopt"].(M)["domainStrategy"] != "ForceIPv4" {
			t.Fatal("missing default")
		}
	}
}
func TestSocketDNSRejectsMalformedAndUnboundedStreams(t *testing.T) {
	if FailClosedSocketDNS(M{"sockopt": []any{}}) == nil {
		t.Fatal("accepted array socket")
	}
	root := M{}
	current := root
	for i := 0; i < 20; i++ {
		child := M{}
		current["xhttpSettings"] = M{"downloadSettings": child}
		current = child
	}
	if FailClosedSocketDNS(root) == nil {
		t.Fatal("accepted unbounded nesting")
	}
}
