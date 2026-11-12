"""Explicit, versioned service drivers. Shared CDNs are intentionally not claimed."""
VERSION = 'service-drivers.v2'
SERVICES = {
 'instagram': {'domains':['instagram.com','cdninstagram.com'], 'probes':[('https://www.instagram.com/robots.txt',200)]},
 'telegram': {'domains':['telegram.org','t.me','telegra.ph'], 'probes':[('https://telegram.org/',200),('https://web.telegram.org/',200)]},
 'x': {'domains':['x.com','twitter.com','twimg.com'], 'probes':[('https://x.com/robots.txt',200)]},
 'whatsapp': {'domains':['whatsapp.com','whatsapp.net'], 'probes':[('https://web.whatsapp.com/',200)]},
 'youtube': {'domains':['youtube.com','youtu.be','googlevideo.com','ytimg.com','youtubei.googleapis.com'], 'probes':[('https://www.youtube.com/generate_204',204),('https://i.ytimg.com/vi/jNQXAC9IVRw/hqdefault.jpg',200)]},
 'google': {'domains':['google.com','gstatic.com','googleapis.com','googleusercontent.com','gvt1.com','gvt2.com'], 'probes':[('https://www.gstatic.com/generate_204',204),('https://play.google.com/robots.txt',200)]},
 'speedtest': {'domains':['speed.cloudflare.com','speedtest.net','ooklaserver.net'], 'probes':[('https://speed.cloudflare.com/__down?bytes=65536',200)]},
 'default': {'domains':[], 'probes':[('https://www.gstatic.com/generate_204',204)]},
}
# Domain rules cannot reliably recognize MTProto IPs, game UDP, shared-CDN
# streaming, encrypted DNS/ECH or WhatsApp calls. These use the measured default.
# Gaming needs a configured controlled UDP echo driver, never DNS as a surrogate.
UNMEASURED = ['authenticated_service_upload','app_media_and_calls',
              'gaming_udp_jitter_and_loss','streaming_playback_and_rebuffering']
