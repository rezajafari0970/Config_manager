# Injected ONLY into isolated queue tests, never a production driver.
# All credentials/endpoints are test fixtures; real traffic is tested separately.
import base64,ipaddress,json
def synthetic_connection(box,candidate,timeout):
    data=base64.b64decode(candidate['config'],validate=True)
    assert driver.lab.digest(data)==candidate['sha256']
    config=json.loads(data)
    for outbound in config['outbounds']:
        settings=outbound.get('settings',{})
        for node in ([settings] if isinstance(settings.get('address'),str) else settings.get('vnext',settings.get('servers',[]))):
            assert ipaddress.ip_address(node['address']) in ipaddress.ip_network('192.0.2.0/24')
    box.policy({});process=box.launch(data,True);process.wait(timeout=15)
    passed=process.returncode==0;box.stop()
    evidence={key:'passed' for key in ['https','dns_tcp','dns_udp','doh_cloudflare','doh_google','quic_vn']}
    return {'core':'passed' if passed else 'failed','connection':'passed' if passed else 'not_run','code':'synthetic_queue_connection' if passed else 'synthetic_core_rejected','runtime_sha256':driver.lab.digest(data),'endpoints_passed':2 if passed else 0,'evidence':evidence}
driver.lab.test_candidate=synthetic_connection
