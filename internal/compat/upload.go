package compat

import (
	"bytes"
	"encoding/json"
	"errors"
)

// UploadDriver is an experimental, opt-in alternative, never a health receipt.
// Promotion requires same-network/core/config A/B evidence and preserved UDP.
// Official 26.11.9 promotes concurrency=1 to128; 26.10.31 retains1.
// This is NOT an effective-concurrency-one guarantee on the latest core.
const UploadDriver = "legacy-vless-tcp-mux-native-udp.experimental.v1"

// UploadAlternative preserves the original bytes and server settings. It only
// proposes TCP mux for plaintext VLESS HTTP-header streams without existing
// active multiplexing, flow or packet-encoding requirements. Native UDP stays
// outside mux, including UDP/443. The remote must actually support mux.cool.
func UploadAlternative(raw []byte) ([]byte, []Change, error) {
	if len(raw) > 2*1024*1024 || !json.Valid(raw) {
		return nil, nil, errors.New("upload_invalid_json")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var root M
	if d.Decode(&root) != nil || root == nil {
		return nil, nil, errors.New("upload_invalid_object")
	}
	nodes, ok := root["outbounds"].([]any)
	if !ok {
		return nil, nil, errors.New("upload_missing_outbounds")
	}
	byTag := map[string]M{}
	for _, value := range nodes {
		ob, ok := value.(M)
		if !ok {
			return nil, nil, errors.New("upload_invalid_outbound")
		}
		if tag, ok := ob["tag"].(string); ok && tag != "" {
			if byTag[tag] != nil {
				return nil, nil, errors.New("upload_duplicate_tag")
			}
			byTag[tag] = ob
		}
	}
	changes := []Change{}
	for _, v := range nodes {
		ob, ok := v.(M)
		if !ok {
			return nil, nil, errors.New("upload_invalid_outbound")
		}
		if !plain(ob) {
			continue
		}
		stream, ok := ob["streamSettings"].(M)
		if !ok {
			continue
		}
		if security, exists := stream["security"]; exists && security != "" && security != "none" {
			continue
		}
		network, _ := stream["network"].(string)
		if method, ok := stream["method"].(string); ok && method != "" {
			network = method
		}
		if network != "" && network != "tcp" && network != "raw" {
			continue
		}
		tcp, _ := stream["rawSettings"].(M)
		if tcp == nil {
			tcp, _ = stream["tcpSettings"].(M)
		}
		header, _ := tcp["header"].(M)
		if header["type"] != "http" || stream["finalmask"] != nil || ob["proxySettings"] != nil {
			continue
		}
		if ref := socksOf(ob)["dialerProxy"]; ref != nil {
			tag, ok := ref.(string)
			if !ok {
				continue
			}
			leg := byTag[tag]
			settings, _ := leg["settings"].(M)
			redirect, _ := settings["redirect"].(string)
			if leg["protocol"] != "freedom" || redirect == "" {
				continue
			}
		}
		ep, err := getEndpoint(ob)
		if err != nil {
			continue
		}
		if enc := ep.user["encryption"]; enc != nil && enc != "" && enc != "none" {
			continue
		}
		if flow := ep.user["flow"]; flow != nil && flow != "" {
			continue
		}
		settings, _ := ob["settings"].(M)
		if encoding := settings["packetEncoding"]; encoding != nil && encoding != "" && encoding != "none" {
			continue
		}
		if mux, exists := ob["mux"]; exists && mux != nil {
			m, ok := mux.(M)
			if !ok {
				return nil, nil, errors.New("upload_invalid_mux")
			}
			if enabled, exists := m["enabled"]; exists && enabled != false {
				continue
			}
		}
		ob["mux"] = M{"enabled": true, "concurrency": 1, "xudpConcurrency": -1, "xudpProxyUDP443": "skip"}
		tag, _ := ob["tag"].(string)
		changes = append(changes, Change{UploadDriver, tag, "experimental_tcp_mux_native_udp_requires_runtime_proof"})
	}
	if len(changes) == 0 {
		return bytes.Clone(raw), changes, nil
	}
	out, err := json.Marshal(root)
	return out, changes, err
}
