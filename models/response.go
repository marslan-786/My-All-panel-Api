package models

// ایس ایم ایس اور میسجز کے لیے کلین اسٹرکچر
type MessageResponse struct {
	DateTime    string `json:"Date-and-time"`
	Number      string `json:"Number"`
	Service     string `json:"Service"`
	FullMessage string `json:"Full_message"`
}

// نمبرز کے لیے کلین اسٹرکچر
type NumberResponse struct {
	Range   string `json:"Range"`
	Country string `json:"Country"`
}

// اگر پیچھے سے کوئی ایرر آئے تو ہمارا سسٹم کریش نہیں ہوگا، بلکہ یہ رسپانس دے گا
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
