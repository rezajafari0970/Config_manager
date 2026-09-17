RULEPACK_VERSION='2026.09.17.1'
RULES={
 'XRAY_SETTINGS_NOT_OBJECT':{'action':'repair','confidence':0.995,'enabled':True},
 'XRAY_STATS_NOT_OBJECT':{'action':'repair','confidence':0.995,'enabled':True},
 'SS_OPAQUE_USERINFO':{'action':'preserve','confidence':0.99,'enabled':True},
 'VLESS_NONSTANDARD_ID':{'action':'preserve','confidence':0.99,'enabled':True},
}
def policy(code): return RULES.get(code,{'action':'observe','confidence':0.5,'enabled':False})
