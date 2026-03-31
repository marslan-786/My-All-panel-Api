# === پہلا مرحلہ (Builder Stage) ===
FROM golang:1.25-bookworm AS go-builder

WORKDIR /app

# سارا کوڈ (main.go, models/, panels/ وغیرہ) کنٹینر میں کاپی کریں
COPY . .

# 1. ڈوکر خود کنٹینر کے اندر go.mod بنائے گا
RUN go mod init api-system

# 2. ڈوکر خود ساری امپورٹس چیک کرے گا اور جو بھی لیٹسٹ لائبریریاں درکار ہوں گی وہ ڈاؤن لوڈ کر لے گا
RUN go mod tidy

# 3. اب کلین طریقے سے پروجیکٹ بلڈ ہوگا
RUN CGO_ENABLED=0 GOOS=linux go build -o api-system .


# === دوسرا مرحلہ (Final Stage) ===
FROM debian:bookworm-slim

# API کالز کے لیے SSL/TLS سرٹیفکیٹس لازمی ہیں
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# بلڈر سٹیج سے صرف فائنل تیار شدہ فائل کاپی کریں
COPY --from=go-builder /app/api-system .

EXPOSE 8080

CMD ["./api-system"]
