#!/usr/bin/env python3
"""Read-only measurement of first assessments in the real production queue.
No source, job, lease, eligibility or worker settings are mutated. Report actual
wall time and first-result states; inconclusive is never counted as definitive.
"""
import argparse, collections, hashlib, json, pathlib, sqlite3, time


def observe(database, count, seconds, output):
    db = sqlite3.connect('file:' + str(pathlib.Path(database).resolve()) + '?mode=ro', uri=True)
    db.execute('PRAGMA query_only=ON')
    begin = time.monotonic()
    begin_ms = int(time.time() * 1000)
    db.execute('BEGIN')
    cursor = db.execute('SELECT coalesce(max(id),0) FROM egress_events').fetchone()[0]
    pending = {r[0] for r in db.execute("SELECT hash FROM egress_jobs WHERE tested=0 AND state!='DELETED' AND paused=0")}
    db.commit()
    if len(pending) < count:
        raise ValueError('insufficient_pending_inputs')
    initial = {}; all_hashes = set(); timeline = []; at600 = None
    while time.monotonic() - begin < seconds:
        for event_id, hash_, at, state in db.execute('SELECT id,hash,at,state FROM egress_events WHERE id>? ORDER BY id', (cursor,)):
            cursor = event_id
            if at < begin_ms:
                continue
            all_hashes.add(hash_)
            if hash_ in pending and hash_ not in initial:
                initial[hash_] = (at, state)
                if len(initial) == count:
                    break
        elapsed = time.monotonic() - begin
        if elapsed >= 600 and at600 is None:
            at600 = sum(at <= begin_ms + 600000 for at, _ in initial.values())
        if not timeline or elapsed - timeline[-1]['seconds'] >= 30 or len(initial) >= count:
            row = dict(seconds=round(elapsed, 3), first_assessments=len(initial), distinct_total=len(all_hashes))
            timeline.append(row); print(json.dumps(row), flush=True)
        if len(initial) >= count:
            break
        time.sleep(1)
    completion = (max((at for at, _ in initial.values()), default=begin_ms) - begin_ms) / 1000
    states = dict(collections.Counter(s for _, s in initial.values()))
    missing = db.execute('SELECT count(DISTINCT m.raw_hash) FROM memberships m WHERE NOT EXISTS(SELECT 1 FROM egress_jobs j WHERE j.hash=m.raw_hash)').fetchone()[0]
    db.close()
    receipt = dict(mode='read_only_live_queue', requested=count, first_assessments=len(initial), seconds=completion,
                   observation_seconds=round(time.monotonic()-begin, 3), assessed_at_600s=at600 if at600 is not None else len(initial),
                   target_600s_met=len(initial) >= count and completion <= 600, states=states,
                   definitive=states.get('RESIDENTIAL', 0)+states.get('NO_SPLIT', 0), initial_pending=len(pending),
                   pending_snapshot_sha256=hashlib.sha256('\n'.join(sorted(pending)).encode()).hexdigest(),
                   unregistered_current=missing, timeline=timeline,
                   scope='First assessments of distinct real pending inputs under normal production scheduling; UNKNOWN/UNSUPPORTED are not conclusive route decisions.')
    pathlib.Path(output).write_text(json.dumps(receipt, indent=2))
    print(json.dumps({k:v for k,v in receipt.items() if k!='timeline'}), flush=True)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--panel-config', default='/etc/config-repair/panel.json')
    parser.add_argument('--count', type=int, default=1000)
    parser.add_argument('--seconds', type=int, default=1200)
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    if not 1 <= args.count <= 2000 or not 10 <= args.seconds <= 3600:
        raise SystemExit('invalid measurement bounds')
    cfg = json.loads(pathlib.Path(args.panel_config).read_text())
    observe(cfg['database'], args.count, args.seconds, args.output)
