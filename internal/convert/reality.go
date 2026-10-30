package convert

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
)

// Validate authentication in retained chains/pools and nested download streams.
// Never infer a key from SNI or weaken REALITY to TLS/plaintext.
func validateReality(ob M) error {
	var visit func(any) error
	visit = func(value any) error {
		switch v := value.(type) {
		case M:
			if v["security"] == "reality" {
				settings, _ := v["realitySettings"].(M)
				key, _ := settings["password"].(string)
				if key == "" {
					key, _ = settings["publicKey"].(string)
				}
				if key == "" {
					return errors.New("missing_reality_public_key")
				}
				decoded, err := base64.RawURLEncoding.DecodeString(key)
				if err != nil || len(decoded) != 32 {
					return errors.New("invalid_reality_public_key")
				}
				if sid, ok := settings["shortId"].(string); ok {
					if _, err := hex.DecodeString(sid); err != nil || len(sid) > 16 {
						return errors.New("invalid_reality_short_id")
					}
				}
			}
			for _, child := range v {
				if err := visit(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range v {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(ob["streamSettings"])
}
