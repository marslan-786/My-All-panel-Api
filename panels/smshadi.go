package panels

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"api-system/models" // یہ آپ کے مین پراجیکٹ کا پاتھ ہونا چاہیے جہاں response.go رکھی ہے
)

// SMSHadi پینل کا مین اسٹرکچر
type SMSHadi struct {
	Client     *http.Client
	Username   string
	Password   string
	SessKey    string
	SMSCache   []models.MessageResponse
	CacheMutex sync.RWMutex
	BaseURL    string
	IsLoggedIn bool
}

// پینل کو انیشلائز کرنے کا فنکشن
func NewSMSHadi(username, password string) *SMSHadi {
	jar, _ := cookiejar.New(nil)
	return &SMSHadi{
		Client:   &http.Client{Jar: jar, Timeout: 20 * time.Second}, // ٹائم آؤٹ تھوڑا بڑھا دیا تاکہ ریکویسٹ فیل نہ ہو
		Username: username,
		Password: password,
		BaseURL:  "http://185.2.83.39/ints",
	}
}

// 1. لاگ ان سسٹم اور کیپچا بائی پاس
func (h *SMSHadi) Login() error {
	log.Println("[SMSHadi] لاگ ان پیج لوڈ ہو رہا ہے...")

	// لاگ ان پیج کو GET کریں تاکہ کیپچا پڑھ سکیں
	req, _ := http.NewRequest("GET", h.BaseURL+"/login", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36")
	resp, err := h.Client.Do(req)
	if err != nil {
		return fmt.Errorf("لاگ ان پیج لوڈ کرنے میں ایرر: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	// Regex سے میتھ کیپچا نکالنا (مثلاً "What is 9 + 6 = ?")
	re := regexp.MustCompile(`What is (\d+) \+ (\d+) = \?`)
	matches := re.FindStringSubmatch(bodyString)

	captchaAns := "0"
	if len(matches) == 3 {
		num1, _ := strconv.Atoi(matches[1])
		num2, _ := strconv.Atoi(matches[2])
		captchaAns = strconv.Itoa(num1 + num2)
	}

	log.Printf("[SMSHadi] کیپچا حل کر لیا گیا: %s, لاگ ان ہو رہا ہے...", captchaAns)

	// POST ریکویسٹ لاگ ان کے لیے
	formData := url.Values{}
	formData.Set("username", h.Username)
	formData.Set("password", h.Password)
	formData.Set("capt", captchaAns)

	postReq, _ := http.NewRequest("POST", h.BaseURL+"/signin", strings.NewReader(formData.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36")
	postReq.Header.Set("Referer", h.BaseURL+"/login")

	postResp, err := h.Client.Do(postReq)
	if err != nil {
		return fmt.Errorf("لاگ ان ریکویسٹ بھیجنے میں ایرر: %v", err)
	}
	defer postResp.Body.Close()

	h.IsLoggedIn = true
	log.Println("[SMSHadi] لاگ ان کامیاب! اب SessKey نکال رہے ہیں...")

	// لاگ ان ہوتے ہی SessKey نکال لو
	err = h.fetchSessKey()
	if err != nil {
		h.IsLoggedIn = false
		return fmt.Errorf("SessKey نکالنے میں ایرر: %v", err)
	}

	return nil
}

// 2. پیج سے SessKey نکالنا
func (h *SMSHadi) fetchSessKey() error {
	req, _ := http.NewRequest("GET", h.BaseURL+"/agent/SMSCDRReports", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36")

	resp, err := h.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	// Regex کے ذریعے sesskey ڈھونڈنا
	re := regexp.MustCompile(`sesskey=([^&"]+)`)
	matches := re.FindStringSubmatch(bodyString)

	if len(matches) > 1 {
		h.SessKey = matches[1]
		log.Printf("[SMSHadi] SessKey مل گئی: %s\n", h.SessKey)
		return nil
	}

	return fmt.Errorf("sesskey نہیں ملی")
}

// 3. بیک گراؤنڈ سروس (جو ہر 5 سیکنڈ بعد چلے گی)
func (h *SMSHadi) StartSMSPoller() {
	go func() {
		for {
			if !h.IsLoggedIn || h.SessKey == "" {
				err := h.Login()
				if err != nil {
					log.Println("[SMSHadi] لاگ ان یا SessKey کا مسئلہ:", err)
					time.Sleep(5 * time.Second)
					continue
				}
			}

			h.fetchAndUpdateSMS()
			time.Sleep(5 * time.Second) // آپ کی ڈیمانڈ کے مطابق 5 سیکنڈ کا ڈیلے
		}
	}()
}

// 4. پینل سے SMS لانا (3 دن کی ڈیٹ کے ساتھ)
func (h *SMSHadi) fetchAndUpdateSMS() {
	now := time.Now()
	// ایک کل (Yesterday), ایک آج, اور ایک آنے والا کل (Tomorrow)
	fdate1 := now.AddDate(0, 0, -1).Format("2006-01-02") + " 00:00:00"
	fdate2 := now.AddDate(0, 0, 1).Format("2006-01-02") + " 23:59:59"

	urlStr := fmt.Sprintf("%s/agent/res/data_smscdr.php?fdate1=%s&fdate2=%s&sesskey=%s&sEcho=1",
		h.BaseURL,
		url.QueryEscape(fdate1),
		url.QueryEscape(fdate2),
		h.SessKey,
	)

	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := h.Client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		log.Println("[SMSHadi] SMS فیچ کرنے میں مسئلہ، سیشن ری سیٹ کر رہے ہیں...")
		h.IsLoggedIn = false // تاکہ اگلی بار خود لاگ ان کر لے
		return
	}
	defer resp.Body.Close()

	var result struct {
		AaData [][]interface{} `json:"aaData"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("[SMSHadi] JSON پارس کرنے میں ایرر:", err)
		return
	}

	var cleanMessages []models.MessageResponse
	for _, row := range result.AaData {
		// کم از کم 6 انڈیکس ہونے چاہئیں تاکہ آؤٹ آف باؤنڈ کا ایرر نہ آئے
		if len(row) > 5 {
			dateStr, ok1 := row[0].(string)
			numberStr, ok2 := row[2].(string)
			serviceStr, ok3 := row[3].(string)
			fullMsgStr, ok4 := row[5].(string)

			// اگر ٹائپ کاسٹنگ ٹھیک نہیں ہوئی یا یہ سمری والی آخری لائن ہے تو اسے چھوڑ دیں
			if !ok1 || !ok2 || !ok3 || !ok4 || strings.Contains(dateStr, ",") {
				continue
			}

			cleanMessages = append(cleanMessages, models.MessageResponse{
				DateTime:    dateStr,
				Number:      numberStr,
				Service:     serviceStr,
				FullMessage: strings.TrimSpace(fullMsgStr),
			})
		}
	}

	// کیش اپڈیٹ کر رہے ہیں (Mutex کے ساتھ تاکہ کوئی ریس کنڈیشن نہ بنے)
	h.CacheMutex.Lock()
	h.SMSCache = cleanMessages
	h.CacheMutex.Unlock()
}

// 5. کلائنٹ کے لیے SMS کا فنکشن (یہ سیدھا کیش سے ڈیٹا دے گا)
func (h *SMSHadi) GetSMSData() []models.MessageResponse {
	h.CacheMutex.RLock()
	defer h.CacheMutex.RUnlock()
	return h.SMSCache
}

// 6. نمبرز والا سسٹم (یہ کلائنٹ کی ریکویسٹ پر ڈائریکٹ پینل سے آئے گا)
func (h *SMSHadi) GetNumbers() ([]models.NumberResponse, error) {
	if !h.IsLoggedIn {
		if err := h.Login(); err != nil {
			return nil, err
		}
	}

	urlStr := fmt.Sprintf("%s/agent/res/data_smsnumbers.php?sEcho=1", h.BaseURL)
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		AaData [][]interface{} `json:"aaData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("نمبرز کا JSON پارس نہیں ہوا: %v", err)
	}

	var cleanNumbers []models.NumberResponse
	for _, row := range result.AaData {
		if len(row) > 3 {
			rangeStr, ok := row[1].(string)
			if !ok {
				continue
			}
			
			// اس پینل میں Range میں ہی ملک کا نام آ رہا ہے (جیسے "Pakistan G 2001")
			cleanNumbers = append(cleanNumbers, models.NumberResponse{
				Range:   rangeStr,
				Country: rangeStr, 
			})
		}
	}

	return cleanNumbers, nil
}
