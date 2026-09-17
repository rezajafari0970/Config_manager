POLICY={
 'initial_retry_seconds':30,
 'healthy_recheck_seconds':300,
 'require_upload':True,
 'require_download':True,
 'external_multi_provider':True,
 'server_traffic_probe':True,
 'healthy_only_if_both_directions':True,
 'delete_after_second_initial_failure':True,
 'delete_on_recheck_failure':True,
 'location_after_health_only':True,
 'remark_format':'{flag} {country_name}',
 'cdn_classification':True,
}
def accepted(upload_ok,download_ok):return bool(upload_ok and download_ok)
def initial_action(attempt,upload_ok,download_ok):
 if accepted(upload_ok,download_ok):return 'promote_healthy'
 return 'retry_after_30s' if attempt==1 else 'delete'
def recheck_action(upload_ok,download_ok):return 'keep' if accepted(upload_ok,download_ok) else 'delete'