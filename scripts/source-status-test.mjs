import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
const source=await readFile(new URL('../internal/panel/web/source-status.js',import.meta.url),'utf8');
const {sourceStatus}=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
const base={enabled:true,state:'updated',config_count:0,content_state:'config',received_count:31,suppressed_count:0};
for(const state of ['updated','unchanged','empty','invalid','filtered'])assert.notEqual(sourceStatus({...base,state})[1],'good');
assert.match(sourceStatus({...base,state:'filtered',suppressed_count:31})[0],/حذف/);
assert.equal(sourceStatus({...base,config_count:31})[1],'good');
assert.equal(sourceStatus({...base,enabled:false,config_count:31})[0],'متوقف');
assert.equal(sourceStatus({...base,state:'error',config_count:31})[1],'wait');
assert.match(sourceStatus({...base,state:'error',config_count:31})[0],/قبلی موجودند/);
assert.match(sourceStatus({...base,state:'error',error_code:'processing_timeout'})[0],/پردازش/);
assert.equal(sourceStatus({...base,state:'error',error_code:'fetch_timeout'})[1],'bad');
assert.equal(sourceStatus({...base,state:'fetching'})[1],'working');
assert.equal(sourceStatus({...base,config_count:29,suppressed_count:1})[1],'wait');
assert.equal(sourceStatus({...base,config_count:1},{blocked:1})[1],'wait');
assert.equal(sourceStatus({...base,config_count:1,content_state:'partial'},{total:1,ready:1})[1],'good');
assert.equal(sourceStatus({...base,config_count:1},{total:1,ready:0,pending:1})[1],'wait');
console.log('14 source status assertions passed; zero current configs never show collection success');
const {receiptCounts}=await import('data:text/javascript;base64,'+Buffer.from(source).toString('base64'));
assert.deepEqual(receiptCounts({received_count:6,receipt_unique:2,receipt_new:0,suppressed_count:0}),{unique:2,repeated:4,added:0,existing:2});
assert.deepEqual(receiptCounts({received_count:6,receipt_unique:2,receipt_new:1,suppressed_count:1}),{unique:2,repeated:4,added:1,existing:0});
assert.equal(receiptCounts({received_count:6,duplicates:6}).unique,null);
assert.match(sourceStatus({...base,stored_count:6})[0],/قبلی/);
assert.match(sourceStatus({...base,state:'queued',refresh_reason:'retention'})[0],/دریافت تازه/);
assert.match(sourceStatus({...base,config_count:2,disabled_count:2})[0],/متوقف/);
console.log('Receipt dedup, unknown legacy measurements, expiry and disabled inventory passed');

const diagnosticSource=await readFile(new URL('../internal/panel/web/diagnostics.js',import.meta.url),'utf8');
const {conversionCodes,conversionDetail,evidenceRows}=await import('data:text/javascript;base64,'+Buffer.from(diagnosticSource).toString('base64'));
for(const code of ['ipv6_only_probe_pending','endpoint_dns_failed','tls_name_mismatch','peer_closed_connection','proxy_request_failed','test_destination_partial','transport_verification_incomplete','unknown_vmess_field','wireguard_extension_unsupported','legacy_vless_transport_requires_driver','transport_driver_pending','concatenated_config_uri']){
 assert.ok(conversionCodes[code],code);assert.notEqual(conversionDetail(code).message,code);
}
assert.equal(conversionDetail('ipv6_only_probe_pending').category,'environment');
assert.equal(conversionDetail('unknown_vmess_field').category,'capability');
assert.equal(conversionDetail('tls_name_mismatch').category,'server');
assert.equal(conversionDetail('transport_verification_incomplete').category,'network');
assert.equal(conversionDetail('unknown-future-code').category,'input');
const rows=evidenceRows({checks:{https:'passed',quic_vn:'failed',dns_udp:'unsupported',android:'not_run'}});
assert.equal(rows.find(r=>r.key==='https').status,'موفق');
assert.equal(rows.find(r=>r.key==='quic_vn').status,'ناموفق');
assert.equal(rows.find(r=>r.key==='dns_udp').status,'پشتیبانی نمی‌شود');
assert.equal(rows.find(r=>r.key==='android').status,'اجرا نشده');
assert.equal(rows.find(r=>r.key==='http3').status,'نامشخص');
assert.equal(evidenceRows({checks:{https:'unexpected'}}).find(r=>r.key==='https').status,'نامشخص');
console.log('Shared diagnostic classification and independent transport evidence passed');

assert.equal(conversionDetail('https_proxy_verified_site_unavailable').category,'network');

for (const state of ['error','queued','fetching']) {const got=sourceStatus({...base,state,config_count:148,refresh_reason:'source_count_drop_pending'});assert.equal(got[1],'wait');assert.match(got[0],/افت تعداد/)}
assert.equal(receiptCounts({received_count:6,receipt_unique:2,receipt_new:-1}).added,null);
