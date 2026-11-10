package panel

import (
	"context"
	"mime"
	"net/http"
	"strings"
)

// Availability is a read-only explanation, shared by all Smart repositories.
// It never claims a build, consumes a delivery turn, or asserts a live connection.
type SmartAvailability struct {
	Available  bool   `json:"available"`
	State      string `json:"state"`
	Message    string `json:"message"`
	Inputs     int    `json:"inputs"`
	Failed     int    `json:"failed"`
	Pending    int    `json:"pending"`
	RetryAfter int    `json:"retry_after"`
}

func (p *SmartPipeline) availability(ctx context.Context) (SmartAvailability, error) {
	v := SmartAvailability{RetryAfter: 15}
	var enabled, worker, eligible, main bool
	scope := p.splash || p.ads
	// One statement observes one committed DB/time snapshot. Only current, enabled,
	// unexpired, unblocked memberships belong to this diagnostic input population.
	e := p.reader().QueryRowContext(ctx, `WITH pool AS MATERIALIZED (
 SELECT DISTINCT m.raw_hash FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id
 WHERE s.enabled=1 AND s.splash=? AND a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)))
 SELECT (SELECT enabled FROM smart_settings WHERE id=1),EXISTS(SELECT 1 FROM smart_ready),
 EXISTS(SELECT 1 FROM smart_worker WHERE id=1 AND status='running' AND heartbeat>CAST(unixepoch('subsec')*1000 AS INTEGER)-30000),
 EXISTS(SELECT 1 FROM smart_offers o JOIN sources src ON src.id=o.source WHERE src.splash=?),
 (SELECT count(*) FROM pool),
 (SELECT count(*) FROM pool JOIN conversion_results r ON r.hash=pool.raw_hash WHERE r.core IN('failed','blocked') OR r.connection IN('failed','partial') OR r.code='https_proxy_verified_site_unavailable'),
 (SELECT count(*) FROM pool LEFT JOIN conversion_jobs j ON j.hash=pool.raw_hash WHERE j.hash IS NULL OR j.state IN('queued','running'))`, scope, scope).Scan(&enabled, &v.Available, &worker, &eligible, &v.Inputs, &v.Failed, &v.Pending)
	if e != nil {
		return v, e
	}
	// Ads followers are ordinary inputs, independent of the Splash head population.
	if p.ads {
		e = p.Store.reader().QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM smart_offers)").Scan(&main)
		if e != nil {
			return v, e
		}
	}
	switch {
	case v.Available:
		v.State = "ready"
		v.Message = "خروجی آماده است."
	case !enabled:
		v.State = "disabled"
		v.Message = "ساخت این نوع اسمارت در تنظیمات غیرفعال است."
	case v.Inputs == 0:
		v.State = "no_inputs"
		v.Message = "ورودی فعال و منقضی‌نشده برای این نوع اسمارت وجود ندارد. منابع ورودی را بررسی کنید."
	case p.ads && !eligible:
		v.State = "ads_health_unavailable"
		v.Message = "اسپلش ادزِ واجد شرایط سلامت موجود نیست؛ قبولی ساختار هسته به‌تنهایی کافی نیست. نتیجهٔ اتصال هر ورودی در صفحهٔ خروجی مشخص است."
	case !eligible:
		v.State = "health_unavailable"
		v.Message = "هنوز ورودی با تأیید معتبر اتصال عمومی و اینستاگرام موجود نیست. بررسی مجدد خودکار انجام می‌شود؛ زمان آماده‌شدن قطعی نیست."
	case p.ads && !main:
		v.State = "main_health_unavailable"
		v.Message = "مسیر اصلی عضو واجد شرایط سلامت ندارد؛ خروجی ادز پس از آماده‌شدن مسیر اصلی ساخته می‌شود."
	case !worker:
		v.State = "worker_unavailable"
		v.Message = "سرویس ساخت اسمارت پاسخ تازه‌ای نداده است؛ وضعیت سرویس نیاز به بررسی دارد."
	default:
		v.State = "building"
		v.Message = "ورودی سالم وجود دارد؛ ساخت و اعتبارسنجی ترکیب‌ها در صف است. جزئیات در صفحهٔ اسمارت نمایش داده می‌شود."
	}
	return v, nil
}

func wantsHTML(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept"), ",") {
		media, params, e := mime.ParseMediaType(strings.TrimSpace(part))
		if e == nil && media == "text/html" && params["q"] != "0" && params["q"] != "0.0" {
			return true
		}
	}
	return false
}

func (m smartModule) unavailable(w http.ResponseWriter, r *http.Request, v SmartAvailability) {
	w.Header().Set("X-Config-Count", "0")
	w.Header().Set("X-Smart-State", v.State)
	w.Header().Set("Retry-After", "15")
	w.Header().Set("Vary", "Accept")
	if wantsHTML(r) && r.Method != http.MethodHead {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = m.s.template.ExecuteTemplate(w, "waiting.html", struct {
			Status SmartAvailability
			Panel  string
		}{v, m.pipeline.path()})
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(v.Message + "\n"))
	}
}
