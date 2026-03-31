package models

// ایس ایم ایس اور میسجز کے لیے کلین اسٹرکچر
type MessageResponse struct {
	DateTime    string `json:"Date-and-time"`
	Number      string `json:"Number"`
	Service     string `json:"Service"`
	FullMessage string `json:"Full_message"`
}

// نمبرز کے لیے کلین اسٹرکچر (یہاں تبدیلی کی گئی ہے)
type NumberResponse struct {
	Range  string `json:"Range"`
	Number string `json:"Number"` // Country کی جگہ Number کر دیا
}

// ایرر ہینڈلنگ اسٹرکچر
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
