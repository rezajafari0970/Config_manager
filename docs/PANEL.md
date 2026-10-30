# Panel

The passwordless Persian panel has Input / Output / Setting. Input manages independent sources, randomized seconds/minutes intervals and per-source deadlines. Output lists transient shared-template conversions and one JSONL export link. Setting manages source retention/fetching and the conversion validator, and retains clear-all source/config actions. See [CONVERSION.md](CONVERSION.md) for exact export gates, compatibility limits and removal of the former bank.

HTTPS, same-origin CSRF, secret URL redaction and encrypted raw/derived storage remain. Public management is explicitly requested. Private worker operations are Unix-only. Run scripts/panel-browser-test.py for isolated responsive/restart/expiry checks; synthetic browser evidence does not certify a real proxy connection.
