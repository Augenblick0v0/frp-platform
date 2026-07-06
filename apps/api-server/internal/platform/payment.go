package platform

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type epayConfig struct {
	BaseURL   string
	SubmitURL string
	PID       string
	Key       string
	SiteName  string
	PublicURL string
}

func epayFromEnv() epayConfig {
	base := strings.TrimRight(getenv("EPAY_API_BASE", "https://pay.flwi.top"), "/")
	return epayConfig{
		BaseURL:   base,
		SubmitURL: strings.TrimRight(getenv("EPAY_SUBMIT_URL", base+"/submit.php"), "/"),
		PID:       getenv("EPAY_PID", ""),
		Key:       getenv("EPAY_KEY", ""),
		SiteName:  getenv("EPAY_SITE_NAME", "FRP Tunnel Platform"),
		PublicURL: strings.TrimRight(getenv("PUBLIC_BASE_URL", getenv("PANEL_PUBLIC_URL", "http://127.0.0.1:8080")), "/"),
	}
}

func (c epayConfig) enabled() bool { return c.PID != "" && c.Key != "" }

func epayMoney(cents int64) string {
	if cents <= 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func epaySignValues(vals map[string]string, key string) string {
	keys := make([]string, 0, len(vals))
	for k, v := range vals {
		if k == "sign" || k == "sign_type" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+vals[k])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + key))
	return hex.EncodeToString(sum[:])
}

func epaySignedValues(vals map[string]string, key string) map[string]string {
	out := make(map[string]string, len(vals)+2)
	for k, v := range vals {
		out[k] = v
	}
	out["sign"] = epaySignValues(out, key)
	out["sign_type"] = "MD5"
	return out
}

func epayBuildSubmitURL(submitURL string, vals map[string]string) string {
	q := make(url.Values)
	for k, v := range vals {
		q.Set(k, v)
	}
	return submitURL + "?" + q.Encode()
}

func newPaymentOrder(userID int64, plan Plan, payType string) PaymentOrder {
	payType = strings.ToLower(strings.TrimSpace(payType))
	if payType == "" {
		payType = "alipay"
	}
	return PaymentOrder{
		UserID:     userID,
		PlanID:     plan.ID,
		Provider:   "epay",
		OutTradeNo: fmt.Sprintf("FP%d%d", userID, time.Now().UnixNano()),
		PayType:    payType,
		Name:       plan.Name,
		Money:      epayMoney(plan.PriceCents),
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
}

func (s *Server) paymentPlans(w http.ResponseWriter, r *http.Request, u User) {
	plans := s.store.Plans()
	out := make([]Plan, 0, len(plans))
	for _, p := range plans {
		if p.Status == "active" {
			out = append(out, p)
		}
	}
	ok(w, out)
}

func (s *Server) createEpayOrder(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	cfg := epayFromEnv()
	if !cfg.enabled() {
		fail(w, 400, "EPAY_NOT_CONFIGURED", "支付接口未配置")
		return
	}
	var in struct {
		PlanID  int64  `json:"plan_id"`
		PayType string `json:"pay_type"`
	}
	if !decode(w, r, &in) {
		return
	}
	var plan Plan
	for _, p := range s.store.Plans() {
		if p.ID == in.PlanID && p.Status == "active" {
			plan = p
			break
		}
	}
	if plan.ID == 0 {
		handleErr(w, ErrNotFound)
		return
	}
	if plan.PriceCents <= 0 {
		fail(w, 400, "PLAN_NOT_FOR_SALE", "套餐未设置价格")
		return
	}
	order, err := s.store.CreatePaymentOrder(newPaymentOrder(u.ID, plan, in.PayType))
	if err != nil {
		handleErr(w, err)
		return
	}
	vals := epaySignedValues(map[string]string{
		"pid":          cfg.PID,
		"type":         order.PayType,
		"out_trade_no": order.OutTradeNo,
		"notify_url":   cfg.PublicURL + "/api/payments/epay/notify",
		"return_url":   cfg.PublicURL + "/api/payments/epay/return",
		"name":         order.Name,
		"money":        order.Money,
		"sitename":     cfg.SiteName,
	}, cfg.Key)
	order.PayURL = epayBuildSubmitURL(cfg.SubmitURL, vals)
	ok(w, map[string]any{
		"id": order.ID, "out_trade_no": order.OutTradeNo, "pay_type": order.PayType, "name": order.Name,
		"money": order.Money, "status": order.Status, "provider": order.Provider, "pay_url": order.PayURL,
	})
}

func (s *Server) epayNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "fail", 400)
		return
	}
	cfg := epayFromEnv()
	if !cfg.enabled() {
		http.Error(w, "fail", 400)
		return
	}
	vals := map[string]string{}
	for k := range r.Form {
		vals[k] = r.Form.Get(k)
	}
	if vals["pid"] != cfg.PID || vals["sign"] == "" || vals["sign"] != epaySignValues(vals, cfg.Key) {
		http.Error(w, "fail", 400)
		return
	}
	if vals["trade_status"] != "TRADE_SUCCESS" {
		_, _ = w.Write([]byte("success"))
		return
	}
	order, err := s.store.PaymentOrderByOutTradeNo(vals["out_trade_no"])
	if err != nil || order.Provider != "epay" || order.Money != vals["money"] {
		http.Error(w, "fail", 400)
		return
	}
	if _, _, err := s.store.MarkPaymentOrderPaid(order.OutTradeNo, vals["trade_no"]); err != nil {
		http.Error(w, "fail", 400)
		return
	}
	_, _ = w.Write([]byte("success"))
}

func (s *Server) epayReturn(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fail(w, 400, "BAD_REQUEST", err.Error())
		return
	}
	cfg := epayFromEnv()
	vals := map[string]string{}
	for k := range r.Form {
		vals[k] = r.Form.Get(k)
	}
	verified := cfg.enabled() && vals["sign"] != "" && vals["sign"] == epaySignValues(vals, cfg.Key)
	ok(w, map[string]any{"verified": verified, "out_trade_no": vals["out_trade_no"], "trade_no": vals["trade_no"], "status": vals["trade_status"]})
}

func parseMoneyCents(money string) int64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(money), 64)
	if err != nil || f <= 0 {
		return 0
	}
	return int64(f*100 + 0.5)
}
