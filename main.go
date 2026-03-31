package main

import (
	"encoding/json"
	"log"
	"net/http"

	"api-system/models"
	"api-system/panels"
)

// گلوبل ویری ایبل تاکہ تمام راؤٹس اس پینل کو ایکسیس کر سکیں
var smsHadiPanel *panels.SMSHadi

func init() {
	log.Println("سسٹم انیشلائز ہو رہا ہے...")

	// پینل کو انیشلائز کر رہے ہیں
	smsHadiPanel = panels.NewSMSHadi("only_possible", "Impossible")
	
	// بیک گراؤنڈ پولر اسٹارٹ کر دیا جو ہر 5 سیکنڈ بعد کیش اپڈیٹ کرے گا
	smsHadiPanel.StartSMSPoller()
}

func main() {
	// SMS والی API (روٹ میں پینل کا نام smshadi ڈال دیا گیا ہے)
	http.HandleFunc("/hadi/sms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		data := smsHadiPanel.GetSMSData()
		json.NewEncoder(w).Encode(data)
	})

	// نمبرز والی API (یہاں بھی پینل کا نام موجود ہے)
	http.HandleFunc("/hadi/numbers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		data, err := smsHadiPanel.GetNumbers()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ErrorResponse{
				Status:  "error",
				Message: err.Error(),
			})
			return
		}
		
		json.NewEncoder(w).Encode(data)
	})

	// سرور کو پورٹ 8080 پر لائیو کر رہے ہیں
	log.Println("سرور پورٹ 8080 پر کامیابی سے سٹارٹ ہو گیا ہے۔ 🚀")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
