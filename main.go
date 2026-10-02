package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
)

const (
	HOST         = "api19.tiktokv.com"
	DEADLINE     = 5 * int64(time.Second)
	REFRESHRATE  = 250 * time.Millisecond
	progBarWidth = 30
	TWITCH_THR   = 1000
	MAX_THREADS  = 5000
)

var (
	BUILDS = []int{247, 312, 322, 357, 358, 415, 422, 444, 466}
	devColors = []*color.Color{
		color.RGB(120, 200, 220),
		color.RGB(180, 180, 220),
		color.RGB(140, 210, 150),
		color.RGB(220, 190, 120),
		color.RGB(200, 160, 200),
	}
	vrgx = regexp.MustCompile(`(?:/video/|v=)(\d{10,30})`)
)



type DeviceModel struct {
	Model string
	Brand string
	Res   string
	DPI   string
}

var androidModels = []DeviceModel{
	{"SM-F926B", "samsung", "1080x2400", "480"},
	{"SM-G998B", "samsung", "1440x3200", "560"},
	{"SM-G991B", "samsung", "1080x2400", "420"},
	{"SM-A536B", "samsung", "1080x2400", "400"},
	{"SM-A546B", "samsung", "1080x2400", "400"},
	{"SM-S918B", "samsung", "1440x3120", "550"},
	{"SM-G781B", "samsung", "1080x2400", "420"},
	{"Pixel 7", "google", "1080x2400", "440"},
	{"Pixel 6", "google", "1080x2400", "440"},
	{"Pixel 8", "google", "1080x2400", "480"},
	{"Pixel 8 Pro", "google", "1440x3120", "550"},
	{"Pixel 7a", "google", "1080x2400", "440"},
	{"Redmi Note 12", "xiaomi", "1080x2400", "440"},
	{"Redmi Note 11", "xiaomi", "1080x2400", "400"},
	{"Redmi Note 13", "xiaomi", "1080x2400", "440"},
	{"POCO F5", "xiaomi", "1080x2400", "440"},
	{"POCO F6", "xiaomi", "1080x2400", "440"},
	{"POCO X5", "xiaomi", "1080x2400", "440"},
	{"OnePlus 11", "oneplus", "1440x3216", "525"},
	{"OnePlus 10", "oneplus", "1440x3216", "525"},
	{"OnePlus 12", "oneplus", "1440x3216", "525"},
	{"OnePlus 9", "oneplus", "1440x3216", "525"},
	{"OPPO Find X5", "oppo", "1080x2400", "450"},
	{"OPPO Find X6", "oppo", "1080x2400", "450"},
	{"OPPO Reno 8", "oppo", "1080x2400", "450"},
	{"vivo X80", "vivo", "1080x2400", "440"},
	{"vivo X90", "vivo", "1080x2400", "440"},
	{"vivo Y78", "vivo", "1080x2400", "400"},
	{"Honor 70", "honor", "1080x2400", "430"},
	{"Honor 90", "honor", "1080x2400", "430"},
	{"Honor Magic5", "honor", "1080x2400", "430"},
	{"Nothing Phone (2)", "nothing", "1080x2400", "440"},
	{"Nothing Phone (1)", "nothing", "1080x2400", "440"},
	{"Realme GT 2", "realme", "1080x2400", "440"},
	{"Realme 11", "realme", "1080x2400", "400"},
	{"Motorola Edge 40", "motorola", "1080x2400", "440"},
	{"Motorola Razr 40", "motorola", "1080x2400", "440"},
	{"Sony Xperia 1 V", "sony", "1080x2400", "440"},
	{"Sony Xperia 5 V", "sony", "1080x2400", "440"},
	{"Nokia X30", "nokia", "1080x2400", "400"},
	{"Nokia G60", "nokia", "1080x2400", "400"},
}

var iosModels = []DeviceModel{
	{"iPhone14,5", "apple", "1170x2532", "460"},
	{"iPhone14,2", "apple", "1170x2532", "460"},
	{"iPhone14,3", "apple", "1170x2532", "460"},
	{"iPhone14,4", "apple", "1170x2532", "460"},
	{"iPhone14,6", "apple", "1170x2532", "460"},
	{"iPhone14,7", "apple", "1170x2532", "460"},
	{"iPhone14,8", "apple", "1284x2778", "458"},
	{"iPad13,4", "apple", "2048x2732", "320"},
	{"iPad13,1", "apple", "1640x2360", "264"},
	{"iPad13,2", "apple", "1640x2360", "264"},
	{"iPad13,10", "apple", "2388x1668", "264"},
	{"iPad13,11", "apple", "2388x1668", "264"},
}

var laptopModels = []DeviceModel{
	{"MacBookPro18,1", "apple", "2560x1600", "220"},
	{"MacBookPro18,2", "apple", "3024x1964", "254"},
	{"MacBookPro18,3", "apple", "3456x2234", "254"},
	{"MacBookPro18,4", "apple", "3024x1964", "254"},
	{"MacBookAir10,1", "apple", "2560x1600", "227"},
	{"XPS-15-9520", "dell", "1920x1200", "180"},
	{"XPS-13-9310", "dell", "1920x1200", "166"},
	{"ThinkPad-X1-Carbon-Gen10", "lenovo", "1920x1200", "170"},
	{"ThinkPad-X1-Carbon-Gen9", "lenovo", "1920x1200", "170"},
	{"Surface-Laptop-5", "microsoft", "2256x1504", "201"},
	{"Surface-Pro-9", "microsoft", "2688x1920", "267"},
	{"HP-Spectre-x360", "hp", "1920x1280", "166"},
	{"ASUS-ZenBook", "asus", "1920x1200", "166"},
}

var timezones = []string{
	"Europe/Paris", "Europe/London", "Europe/Berlin", "Europe/Rome", "Europe/Madrid", "Europe/Amsterdam",
	"Europe/Stockholm", "Europe/Warsaw", "Europe/Prague", "Europe/Vienna", "Europe/Zurich", "Europe/Brussels",
	"Asia/Kolkata", "Asia/Tokyo", "Asia/Shanghai", "Asia/Hong_Kong", "Asia/Singapore", "Asia/Seoul",
	"Asia/Bangkok", "Asia/Jakarta", "Asia/Manila", "Asia/Kuala_Lumpur", "Asia/Taipei", "Asia/Dubai",
	"America/New_York", "America/Los_Angeles", "America/Chicago", "America/Denver", "America/Phoenix",
	"America/Toronto", "America/Mexico_City", "America/Sao_Paulo", "America/Buenos_Aires", "America/Lima",
	"Africa/Cairo", "Africa/Lagos", "Africa/Johannesburg", "Africa/Nairobi", "Africa/Casablanca",
	"Australia/Sydney", "Australia/Melbourne", "Australia/Perth", "Australia/Brisbane", "Australia/Adelaide",
	"Pacific/Auckland", "Pacific/Fiji", "Pacific/Guam", "Pacific/Honolulu",
}

var regions = []string{
	"US", "CA", "GB", "DE", "FR", "IT", "ES", "NL", "SE", "NO", "DK", "FI", "PL", "CZ", "AT", "CH", "BE",
	"IN", "CN", "JP", "KR", "SG", "TH", "ID", "MY", "PH", "VN", "TW", "HK", "AE", "SA", "IL",
	"BR", "AR", "MX", "CO", "PE", "CL", "VE", "EC", "BO", "UY", "PY",
	"ZA", "EG", "NG", "KE", "MA", "GH", "TN", "DZ", "AO", "ET", "TZ", "UG",
	"AU", "NZ", "FJ", "PG", "GU", "HI", "FJ", "SB", "VU", "NC", "PF",
	"RU", "TR", "UA", "KZ", "UZ", "KG", "TJ", "TM", "GE", "AM", "AZ",
}

type TTDEV struct {
	Did  string `json:"device_id"`
	Iid  string `json:"iid"`
	Dev  string `json:"device_type"`
	Brnd string `json:"device_brand"`
	Osv  string `json:"os_version"`
	Vc   string `json:"version_code"`
	Reg  string `json:"region"`
	Ch   string `json:"channel"`
	App  string `json:"app_name"`
	Ua   string `json:"user_agent"`
}

type tiktokStats struct {
	s, f, t, e atomic.Int64
}

var (
	devPool  []TTDEV
	devIndex atomic.Int64
)

func init() {
	devPool = make([]TTDEV, 50000)
	for i := range devPool {
		devPool[i] = generateTTDEV()
	}
}

func getDevice() TTDEV {
	return devPool[devIndex.Add(1)%int64(len(devPool))]
}

type twitchStats struct {
	s atomic.Int64
	f atomic.Int64
}

func progressStr(sent, total int64) string {
	pct := float64(sent) / float64(total) * 100
	if pct > 100 {
		pct = 100
	}
	return fmt.Sprintf("%d/%d (%.0f%%)", sent, total, pct)
}

func randDigits(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + rand.Intn(10))
	}
	return string(b)
}

func generateUUID() string {
	u := make([]byte, 36)
	for i := range u {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			u[i] = '-'
		} else {
			u[i] = "0123456789abcdef"[rand.Intn(16)]
		}
	}
	return string(u)
}

func generateTokenHex(n int) string {
	h := "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = h[rand.Intn(len(h))]
	}
	return string(b)
}

func generateTTDEV() TTDEV {
	dt := rand.Intn(3)
	var model, brand, osv, ua, ch string

	switch dt {
	case 0:
		s := androidModels[rand.Intn(len(androidModels))]
		model, brand = s.Model, s.Brand
		osv = []string{"10", "11", "12", "13", "14"}[rand.Intn(5)]
		ua = fmt.Sprintf("com.ss.android.ugc.trill/300804 (Linux; U; Android %s; en_US; %s; Build/RP1A.200720.012; Cronet/58.0.2991.0)", osv, model)
		ch = "googleplay"
	case 1:
		s := iosModels[rand.Intn(len(iosModels))]
		model, brand = s.Model, s.Brand
		osv = []string{"14.8", "15.6", "16.2", "17.0"}[rand.Intn(4)]
		ua = fmt.Sprintf("Trill/300804 (iOS %s)", osv)
		ch = "appstore"
	default:
		s := laptopModels[rand.Intn(len(laptopModels))]
		model, brand = s.Model, s.Brand
		osv = []string{"10", "11", "12"}[rand.Intn(3)]
		ua = fmt.Sprintf("Mozilla/5.0 (%s; OS %s)", brand, osv)
		ch = "desktop"
	}

	return TTDEV{
		Did:  randDigits(19),
		Iid:  randDigits(19),
		Dev:  model,
		Brnd: brand,
		Osv:  osv,
		Vc:   "300804",
		Reg:  regions[rand.Intn(len(regions))],
		Ch:   ch,
		App:  "trill",
		Ua:   ua,
	}
}

type deviceJSON struct {
	DeviceID            string `json:"device_id"`
	Iid                 string `json:"iid"`
	DeviceType          string `json:"device_type"`
	DeviceBrand         string `json:"device_brand"`
	Cdid                string `json:"cdid"`
	Openudid            string `json:"openudid"`
	Cookie              string `json:"cookie"`
	OsVersion           string `json:"os_version"`
	VersionCode         string `json:"version_code"`
	VersionName         string `json:"version_name"`
	UpdateVersionCode   string `json:"update_version_code"`
	ManifestVersionCode string `json:"manifest_version_code"`
	Resolution          string `json:"resolution"`
	Dpi                 string `json:"dpi"`
	Aid                 string `json:"aid"`
	Channel             string `json:"channel"`
	AppName             string `json:"app_name"`
	Region              string `json:"region"`
	SysRegion           string `json:"sys_region"`
	CarrierRegion       string `json:"carrier_region"`
	TimezoneName        string `json:"timezone_name"`
	TimezoneOffset      int    `json:"timezone_offset"`
	UserAgent           string `json:"user_agent"`
	DeviceToken         string `json:"device_token"`
	ReportToken         string `json:"report_token"`
}

func generateDevice() string {
	dt := rand.Intn(3)
	var model, brand, res, dpi, osv, ua, ch string

	switch dt {
	case 0:
		s := androidModels[rand.Intn(len(androidModels))]
		model, brand, res, dpi = s.Model, s.Brand, s.Res, s.DPI
		osv = []string{"10", "11", "12", "13", "14"}[rand.Intn(5)]
		ua = fmt.Sprintf("com.ss.android.ugc.trill/300804 (Linux; U; Android %s; en_US; %s; Build/RP1A.200720.012; Cronet/58.0.2991.0)", osv, model)
		ch = "googleplay"
	case 1:
		s := iosModels[rand.Intn(len(iosModels))]
		model, brand, res, dpi = s.Model, s.Brand, s.Res, s.DPI
		osv = []string{"14.8", "15.6", "16.2", "17.0"}[rand.Intn(4)]
		ua = fmt.Sprintf("Trill/300804 (iOS %s)", osv)
		ch = "appstore"
	default:
		s := laptopModels[rand.Intn(len(laptopModels))]
		model, brand, res, dpi = s.Model, s.Brand, s.Res, s.DPI
		osv = []string{"10", "11", "12"}[rand.Intn(3)]
		ua = fmt.Sprintf("Mozilla/5.0 (%s; OS %s)", brand, osv)
		ch = "desktop"
	}

	b, _ := json.Marshal(deviceJSON{
		DeviceID:            randDigits(19),
		Iid:                 randDigits(19),
		DeviceType:          model,
		DeviceBrand:         brand,
		Cdid:                generateUUID() + randDigits(6),
		Openudid:            generateTokenHex(20),
		OsVersion:           osv,
		VersionCode:         "300804",
		VersionName:         "3.0.1.0",
		UpdateVersionCode:   "300804",
		ManifestVersionCode: "3008040",
		Resolution:          res,
		Dpi:                 dpi,
		Aid:                 "1180",
		Channel:             ch,
		AppName:             "trill",
		Region:              regions[rand.Intn(len(regions))],
		SysRegion:           regions[rand.Intn(len(regions))],
		CarrierRegion:       regions[rand.Intn(len(regions))],
		TimezoneName:        timezones[rand.Intn(len(timezones))],
		TimezoneOffset:      rand.Intn(10000),
		UserAgent:           ua,
	})
	return string(b)
}

func cls() {
	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/c", "cls").Stdout = os.Stdout
		exec.Command("cmd", "/c", "cls").Run()
	default:
		fmt.Print("\033[2J\033[H")
	}
}

func startMenu() {
	cls()
	fmt.Println("=== TK FUCKER V2 ===")
	fmt.Println()
	fmt.Println("  [1] TikTok Viewer")
	fmt.Println("  [2] TikTok Sharer")
	fmt.Println("  [3] Twitch Follow Bot")
	fmt.Println("  [4] Automated Storefront Worker (EXST Site Auto-Fulfill)")
	fmt.Println("  [5] Exit")
	fmt.Println()
	fmt.Print("  [?] select: ")
}

func getvid(l string) (string, error) {
	l = strings.TrimSpace(l)
	if matched, _ := regexp.MatchString(`^\d{10,30}$`, l); matched {
		return l, nil
	}
	if !strings.Contains(l, "tiktok.com/") {
		return "", errors.New("not a tiktok link")
	}

	c := &http.Client{Timeout: time.Second * 10, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	}}
	resp, err := c.Get(l)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	m := vrgx.FindStringSubmatch(resp.Request.URL.String())
	if len(m) < 2 {
		return "", errors.New("couldnt find id")
	}
	return m[1], nil
}

func sendview(c *http.Client, vid string, st *tiktokStats) {
	build := BUILDS[rand.Intn(len(BUILDS))]
	d := getDevice()
	osv := rand.Intn(7) + 5

	p := url.Values{
		"app_language": {"fr"}, "iid": {d.Iid}, "device_id": {d.Did},
		"channel": {d.Ch}, "device_type": {d.Dev}, "ac": {"wifi"},
		"os_version": {strconv.Itoa(osv)}, "version_code": {strconv.Itoa(build)},
		"app_name": {d.App}, "device_brand": {d.Brnd}, "ssmix": {"a"},
		"device_platform": {"android"}, "aid": {"1180"}, "as": {"a1iosdfgh"}, "cp": {"androide1"},
	}
	h := map[string]string{
		"Host": HOST, "Connection": "keep-alive", "Accept-Encoding": "gzip",
		"X-SS-REQ-TICKET": strconv.FormatInt(time.Now().UnixMilli(), 10),
		"Content-Type":    "application/x-www-form-urlencoded; charset=UTF-8",
		"User-Agent":      d.Ua,
	}
	b := url.Values{
		"manifest_version_code": {strconv.Itoa(build)},
		"update_version_code":   {strconv.Itoa(build) + "0"},
		"play_delta": {"1"}, "item_id": {vid},
		"version_code": {strconv.Itoa(build)}, "aweme_type": {"0"},
	}

	u := fmt.Sprintf("https://%s/aweme/v1/aweme/stats?%s", HOST, p.Encode())
	req, _ := http.NewRequest("POST", u, bytes.NewBufferString(b.Encode()))
	for k, v := range h {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			st.t.Add(1)
		} else {
			st.e.Add(1)
		}
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("Content-Type"), "charset=utf-8") {
		st.s.Add(1)
	} else {
		st.f.Add(1)
	}
}

func sendshare(c *http.Client, vid string, st *tiktokStats) {
	build := BUILDS[rand.Intn(len(BUILDS))]
	d := getDevice()
	osv := rand.Intn(7) + 5

	p := url.Values{
		"app_language": {"fr"}, "iid": {d.Iid}, "device_id": {d.Did},
		"channel": {d.Ch}, "device_type": {d.Dev}, "ac": {"wifi"},
		"os_version": {strconv.Itoa(osv)}, "version_code": {strconv.Itoa(build)},
		"app_name": {d.App}, "device_brand": {d.Brnd}, "ssmix": {"a"},
		"device_platform": {"android"}, "aid": {"1180"}, "as": {"a1iosdfgh"}, "cp": {"androide1"},
	}
	h := map[string]string{
		"Host": HOST, "Connection": "keep-alive", "Accept-Encoding": "gzip",
		"X-SS-REQ-TICKET": strconv.FormatInt(time.Now().UnixMilli(), 10),
		"Content-Type":    "application/x-www-form-urlencoded; charset=UTF-8",
		"User-Agent":      d.Ua,
	}
	b := url.Values{
		"manifest_version_code": {strconv.Itoa(build)},
		"update_version_code":   {strconv.Itoa(build) + "0"},
		"share_delta": {"1"}, "item_id": {vid},
		"version_code": {strconv.Itoa(build)}, "aweme_type": {"0"},
	}

	u := fmt.Sprintf("https://%s/aweme/v1/aweme/stats?%s", HOST, p.Encode())
	req, _ := http.NewRequest("POST", u, bytes.NewBufferString(b.Encode()))
	for k, v := range h {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			st.t.Add(1)
		} else {
			st.e.Add(1)
		}
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 && strings.Contains(resp.Header.Get("Content-Type"), "charset=utf-8") {
		st.s.Add(1)
	} else {
		st.f.Add(1)
	}
}

type clientPool struct {
	clients []*http.Client
	next    int32
}

func newClientPool(n, conns int, proxy string) *clientPool {
	cp := &clientPool{clients: make([]*http.Client, n)}
	per := (conns + n - 1) / n
	for i := 0; i < n; i++ {
		tp := &http.Transport{
			MaxIdleConns:        per,
			MaxIdleConnsPerHost: per,
			MaxConnsPerHost:     per + 5,
		}
		if proxy != "" {
			if u, err := url.Parse(proxyURL(proxy)); err == nil {
				tp.Proxy = http.ProxyURL(u)
			}
		}
		cp.clients[i] = &http.Client{Timeout: time.Duration(DEADLINE), Transport: tp}
	}
	return cp
}

func (cp *clientPool) Get() *http.Client {
	return cp.clients[atomic.AddInt32(&cp.next, 1)%int32(len(cp.clients))]
}

func runTikTok(vid string, tgt int, thr int) int64 {
	return runTikTokWithProgress(vid, tgt, thr, nil)
}

func runTikTokWithProgress(vid string, tgt int, thr int, onProgress func(int64)) int64 {
	start := time.Now()
	st := &tiktokStats{}
	pool := newClientPool(10, thr, "")

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(REFRESHRATE)
		defer t.Stop()
		lastProgressReport := time.Now()
		for {
			select {
			case <-t.C:
				s := st.s.Load()
				elapsed := time.Since(start).Seconds()
				r := 0.0
				if elapsed > 0 {
					r = float64(s) / elapsed
				}
				fmt.Printf("\r  views: %s  rate: %.0f/s  ", progressStr(s, int64(tgt)), r)
				if onProgress != nil && time.Since(lastProgressReport) >= 1200*time.Millisecond {
					lastProgressReport = time.Now()
					onProgress(s)
				}
			case <-done:
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < thr; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if st.s.Load() >= int64(tgt) {
					return
				}
				sendview(pool.Get(), vid, st)
			}
		}()
	}
	wg.Wait()
	close(done)

	elapsed := time.Since(start).Seconds()
	sent := st.s.Load()
	r := float64(sent) / elapsed
	fmt.Printf("\r  views: %s  rate: %.0f/s\n", progressStr(sent, int64(tgt)), r)
	return sent
}

func runCLIViewer() {
	cls()
	fmt.Println("=== TK FUCKER V2 ===")
	fmt.Println()

	s := bufio.NewScanner(os.Stdin)
	fmt.Print("  [?] url/id: ")
	s.Scan()
	u := strings.TrimSpace(s.Text())

	fmt.Print("  [?] views: ")
	s.Scan()
	v := strings.TrimSpace(s.Text())
	fmt.Println()

	n, err := strconv.Atoi(v)
	if err != nil {
		fmt.Println("  [!] invalid number:", err)
		return
	}

	vid, err := getvid(u)
	if err != nil {
		fmt.Println("  [!]", err)
		return
	}

	fmt.Printf("  [*] sending %d views\n", n)
	sent := runTikTok(vid, n, 3000)
	fmt.Println()
	fmt.Printf("  [+] done — %d views sent\n", sent)
}

func runTikTokShares(vid string, thr int) {
	st := &tiktokStats{}
	pool := newClientPool(10, thr, "")

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		close(stop)
	}()

	var wg sync.WaitGroup
	for i := 0; i < thr; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					sendshare(pool.Get(), vid, st)
				}
			}
		}()
	}
	wg.Wait()

	fmt.Println("\n  [+] done")
}

func runCLISharer() {
	cls()
	fmt.Println("=== TK FUCKER V2 ===")
	fmt.Println()

	s := bufio.NewScanner(os.Stdin)
	fmt.Print("  [?] url/id: ")
	s.Scan()
	u := strings.TrimSpace(s.Text())
	fmt.Println()

	vid, err := getvid(u)
	if err != nil {
		fmt.Println("  [!]", err)
		return
	}

	fmt.Println("  [*] started — press Ctrl+C to stop")
	runTikTokShares(vid, 3000)
}

func readLines(p string) ([]string, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

func proxyURL(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}
	return "http://" + p
}

func gqlClient(proxy string) *http.Client {
	t := &http.Transport{}
	if proxy != "" {
		u, err := url.Parse(proxyURL(proxy))
		if err == nil {
			t.Proxy = http.ProxyURL(u)
		}
	}
	return &http.Client{Timeout: 10 * time.Second, Transport: t}
}

func getTwitchUserID(username, proxy string) (string, error) {
	h := map[string]string{
		"Client-Id":    "kimne78kx3ncx6brgo4mv6wki5h1ko",
		"Content-Type": "application/json",
		"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
	payload, _ := json.Marshal([]map[string]interface{}{{
		"operationName": "GetIDFromLogin",
		"variables":     map[string]string{"login": username},
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version": 1, "sha256Hash": "94e82a7b1e3c21e186daa73ee2afc4b8f23bade1fbbff6fe8ac133f50a2f58ca",
			},
		},
	}})

	c := gqlClient(proxy)
	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", strings.NewReader(string(payload)))
	for k, v := range h {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data []struct {
		Data struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data) == 0 || data[0].Data.User.ID == "" {
		return "", fmt.Errorf("user not found")
	}
	return data[0].Data.User.ID, nil
}

func twitchFollow(targetID, token, proxy string, st *twitchStats) {
	h := map[string]string{
		"Accept":        "application/json",
		"Authorization": "OAuth " + strings.TrimSpace(token),
		"Content-Type":  "application/json",
		"User-Agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
	payload, _ := json.Marshal([]map[string]interface{}{{
		"operationName": "FollowUserMutation",
		"variables": map[string]interface{}{
			"targetId": targetID, "disableNotifications": false,
		},
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version": 1, "sha256Hash": "cd112d9483ede85fa0da514a5657141c24396efbc7bac0ea3623e839206573b8",
			},
		},
	}})

	c := gqlClient(proxy)
	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", strings.NewReader(string(payload)))
	for k, v := range h {
		req.Header.Set(k, v)
	}

	resp, err := c.Do(req)
	if err != nil {
		st.f.Add(1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		st.s.Add(1)
	} else {
		st.f.Add(1)
	}
}

func twitchWorker(sem chan struct{}, targetID string, tgt int, st *twitchStats, wg *sync.WaitGroup, tokens []string, proxy string) {
	defer wg.Done()
	for {
		sem <- struct{}{}
		if st.s.Load() >= int64(tgt) {
			<-sem
			return
		}
		twitchFollow(targetID, tokens[rand.Intn(len(tokens))], proxy, st)
		<-sem
	}
}

func twitchRun(targetID string, tgt int, tokens []string, proxy string) int64 {
	start := time.Now()
	st := &twitchStats{}
	wg := &sync.WaitGroup{}
	sem := make(chan struct{}, TWITCH_THR)

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s := st.s.Load()
				elapsed := time.Since(start).Seconds()
				r := 0.0
				if elapsed > 0 {
					r = float64(s) / elapsed
				}
				fmt.Printf("\r  follows: %s  rate: %.0f/s  ", progressStr(s, int64(tgt)), r)
			case <-done:
				return
			}
		}
	}()

	for i := 0; i < TWITCH_THR; i++ {
		wg.Add(1)
		go twitchWorker(sem, targetID, tgt, st, wg, tokens, proxy)
	}
	wg.Wait()
	close(done)

	elapsed := time.Since(start).Seconds()
	sent := st.s.Load()
	r := float64(sent) / elapsed
	fmt.Printf("\r  follows: %s  rate: %.0f/s\n", progressStr(sent, int64(tgt)), r)
	return sent
}

func runTwitcher() {
	cls()
	fmt.Println("=== TK FUCKER V2 ===")
	fmt.Println()

	s := bufio.NewScanner(os.Stdin)
	fmt.Print("  [?] proxy (user:pass@host:port): ")
	s.Scan()
	px := strings.TrimSpace(s.Text())

	fmt.Print("  [?] target username: ")
	s.Scan()
	un := strings.TrimSpace(s.Text())

	fmt.Print("  [?] amount: ")
	s.Scan()
	as := strings.TrimSpace(s.Text())
	amount, err := strconv.Atoi(as)
	if err != nil {
		fmt.Println("  [!] invalid amount")
		return
	}

	tokens, err := readLines("accounts.txt")
	if err != nil {
		fmt.Println("  [!] accounts.txt:", err)
		return
	}
	if len(tokens) == 0 {
		fmt.Println("  [!] accounts.txt empty")
		return
	}

	fmt.Println("  [*] resolving user id...")
	targetID, err := getTwitchUserID(un, px)
	if err != nil || targetID == "" {
		fmt.Println("  [!] couldn't resolve user")
		return
	}

	fmt.Println("  [+] target id:", targetID)
	fmt.Printf("  [*] sending %d follows\n", amount)

	sent := twitchRun(targetID, amount, tokens, px)
	fmt.Println()
	fmt.Printf("  [+] done — %d follows sent\n", sent)
}

var (
	devPrintLock sync.Mutex
	devFileLock  sync.Mutex
	devCounter   int64
	devAmount    int
)

func deviceWorker(wg *sync.WaitGroup, output *os.File, _ int) {
	defer wg.Done()
	dev := generateDevice()
	devFileLock.Lock()
	output.WriteString(dev + "\n")
	devFileLock.Unlock()

	count := atomic.AddInt64(&devCounter, 1)
	if count%50 == 0 || count == int64(devAmount) {
		devPrintLock.Lock()
		devColors[int(count)%len(devColors)].Printf("\rAmount Devices Generated: %d", count)
		devPrintLock.Unlock()
	}
}

func runDeviceGen() {

	color.RGB(120, 160, 200).Printf("[+] file name to auto make + save devices to ? [+] ")
	var fn string
	fmt.Scanln(&fn)
	if fn == "" {
		color.RGB(220, 100, 100).Printf("[+] Error: Filename cannot be empty.\n")
		return
	}

	os.MkdirAll("saved devices", 0755)
	fp := filepath.Join("saved devices", fn)
	color.RGB(220, 190, 120).Printf("[+] Auto-saving to: %s [+]\n", fp)

	color.RGB(120, 160, 200).Printf("[+] how many devices to generate ? [+] ")
	var as string
	fmt.Scanln(&as)
	devAmount, _ = strconv.Atoi(as)

	output, err := os.Create(fp)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer output.Close()
	w := bufio.NewWriterSize(output, 32*1024*1024)
	defer w.Flush()

	var wg sync.WaitGroup
	sem := make(chan struct{}, MAX_THREADS)
	for i := 0; i < devAmount; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			deviceWorker(&wg, output, 0)
		}()
	}
	wg.Wait()

	fmt.Printf("\n")
	color.RGB(140, 210, 150).Printf("\n[✓] DONE — %d devices\n", devAmount)
	color.RGB(140, 210, 150).Printf("[✓] Saved to: %s\n", fp)
}

type WorkerTask struct {
	OrderID       string `json:"orderId"`
	OrderCode     string `json:"orderCode"`
	TikTokURL     string `json:"tiktokUrl"`
	Quantity      int    `json:"quantity"`
	ViewsPerUnit  int    `json:"viewsPerUnit"`
	TargetViews   int    `json:"targetViews"`
	CustomerEmail string `json:"customerEmail"`
}

type WorkerTasksResponse struct {
	Success bool         `json:"success"`
	Count   int          `json:"count"`
	Tasks   []WorkerTask `json:"tasks"`
	Error   string       `json:"error,omitempty"`
}

func runStoreWorker(siteURL, workerKey string) {
	cls()
	color.RGB(120, 200, 255).Println("==================================================")
	color.RGB(120, 200, 255).Println("     EXST ONLINE - AUTOMATED STOREFRONT WORKER    ")
	color.RGB(120, 200, 255).Println("==================================================")
	fmt.Printf("[*] Site API: %s\n", siteURL)
	fmt.Println("[*] Listening for paid TikTok view orders...")
	fmt.Println("[*] Press Ctrl+C to return / stop worker.")
	fmt.Println()

	client := &http.Client{Timeout: 15 * time.Second}

	for {
		endpoint := strings.TrimRight(siteURL, "/") + "/api/worker/tiktok-views"
		req, err := http.NewRequest("GET", endpoint, nil)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		req.Header.Set("x-worker-key", workerKey)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("\r[!] Worker poll error: %v (retrying in 5s)...    ", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			var res WorkerTasksResponse
			if err := json.Unmarshal(body, &res); err == nil && res.Success && len(res.Tasks) > 0 {
				color.RGB(140, 210, 150).Printf("\n[+] Found %d pending order(s)!\n", len(res.Tasks))
				for _, task := range res.Tasks {
					processWorkerTask(client, endpoint, workerKey, task)
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func processWorkerTask(client *http.Client, endpoint, workerKey string, task WorkerTask) {
	color.RGB(120, 200, 220).Printf("\n[>] Processing Order %s for %s\n", task.OrderCode, task.CustomerEmail)
	fmt.Printf("    Target Link: %s\n", task.TikTokURL)
	fmt.Printf("    Views Target: %d views\n", task.TargetViews)

	// Send processing update
	updateTaskStatus(client, endpoint, workerKey, task.OrderID, "PROCESSING", 0, "")

	vid, err := getvid(task.TikTokURL)
	if err != nil {
		color.RGB(220, 100, 100).Printf("    [!] Error resolving TikTok video ID: %v\n", err)
		updateTaskStatus(client, endpoint, workerKey, task.OrderID, "FAILED", 0, err.Error())
		return
	}

	color.RGB(140, 210, 150).Printf("    [+] Resolved Video ID: %s\n", vid)
	fmt.Printf("    [*] Dispatching %d views now...\n", task.TargetViews)

	sent := runTikTokWithProgress(vid, task.TargetViews, 3000, func(currentSent int64) {
		go updateTaskStatus(client, endpoint, workerKey, task.OrderID, "PROCESSING", currentSent, "")
	})

	color.RGB(140, 210, 150).Printf("    [✓] Order %s complete! Sent %d views.\n", task.OrderCode, sent)
	updateTaskStatus(client, endpoint, workerKey, task.OrderID, "COMPLETED", sent, "")
}

func updateTaskStatus(client *http.Client, endpoint, workerKey, orderID, status string, viewsSent int64, errStr string) {
	payload := map[string]interface{}{
		"orderId":   orderID,
		"status":    status,
		"viewsSent": viewsSent,
	}
	if errStr != "" {
		payload["error"] = errStr
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(b))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-worker-key", workerKey)

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func main() {
	workerFlag := flag.Bool("worker", false, "Run automated storefront worker daemon")
	siteFlag := flag.String("site", "https://exststocks.org", "Storefront URL")
	keyFlag := flag.String("key", "EXST_Crypto_HMAC_Secret_991283_Uncrackable", "Worker secret key")
	urlFlag := flag.String("url", "", "Direct TikTok video URL or ID")
	viewsFlag := flag.Int("views", 0, "Amount of views to send directly")
	flag.Parse()

	siteURL := *siteFlag
	if envSite := os.Getenv("SITE_URL"); envSite != "" {
		siteURL = envSite
	}

	workerKey := *keyFlag
	if envKey := os.Getenv("WORKER_KEY"); envKey != "" {
		workerKey = envKey
	}

	isWorkerMode := *workerFlag || os.Getenv("WORKER") == "true" || os.Getenv("PORT") != ""

	// If running in cloud container with a assigned PORT, start lightweight health probe
	port := os.Getenv("PORT")
	if port != "" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"online","service":"exst-storefront-worker"}`))
			})
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
			_ = http.ListenAndServe(":"+port, mux)
		}()
	}

	if isWorkerMode {
		runStoreWorker(siteURL, workerKey)
		return
	}

	if *urlFlag != "" && *viewsFlag > 0 {
		vid, err := getvid(*urlFlag)
		if err != nil {
			fmt.Println("[!] Error resolving URL:", err)
			return
		}
		fmt.Printf("[*] Sending %d views to %s...\n", *viewsFlag, vid)
		sent := runTikTok(vid, *viewsFlag, 3000)
		fmt.Printf("[✓] Finished: %d views sent.\n", sent)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "gen" {
		runDeviceGen()
		return
	}

	scan := bufio.NewScanner(os.Stdin)
	for {
		startMenu()
		scan.Scan()
		sel := strings.TrimSpace(scan.Text())

		switch sel {
		case "1":
			runCLIViewer()
		case "2":
			runCLISharer()
		case "3":
			runTwitcher()
		case "4":
			runStoreWorker(*siteFlag, *keyFlag)
		case "5":
			cls()
			fmt.Println("bye")
			return
		default:
			fmt.Println("  [!] invalid option")
		}

		fmt.Println()
		fmt.Print("  [?] press enter to return to menu...")
		scan.Scan()
	}
}
