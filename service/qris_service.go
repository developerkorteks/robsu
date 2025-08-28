package service

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QRIS statis original dari merchant
const staticQRIS = "00020101021126610014COM.GO-JEK.WWW01189360091433636775460210G3636775460303UMI51440014ID.CO.QRIS.WWW0215ID10254166023610303UMI5204899953033605802ID5925GIRI RAYA NURSAMTO, Digit6012KOTA CIREBON61054512162070703A016304D5CA"

// GenerateDynamicQRIS membuat QRIS dinamis dengan nominal tertentu
func GenerateDynamicQRIS(amount int64) (string, error) {
	// 1. Hapus 4 karakter terakhir (CRC lama)
	qrisTrim := staticQRIS[:len(staticQRIS)-4]

	// 2. Ganti 010211 -> 010212 (statis -> dinamis)
	qrisTrim = strings.Replace(qrisTrim, "010211", "010212", 1)

	// 3. Pisah di "5802ID"
	parts := strings.Split(qrisTrim, "5802ID")
	if len(parts) != 2 {
		return "", fmt.Errorf("format QRIS tidak valid")
	}

	// 4. Format nominal
	nominal := fmt.Sprintf("%d", amount)
	amountTag := "54" + fmt.Sprintf("%02d", len(nominal)) + nominal

	// 5. Gabungkan kembali
	newQRIS := parts[0] + amountTag + "5802ID" + parts[1]

	// 6. Hitung CRC16 baru
	newQRIS += computeCRC16(newQRIS)

	return newQRIS, nil
}

// GenerateQRCodeImage membuat file QR code PNG
func GenerateQRCodeImage(qrisCode, filename string) error {
	return qrcode.WriteFile(qrisCode, qrcode.Medium, 256, filename)
}

// GenerateQRCodeBytes membuat QR code dalam bentuk bytes untuk dikirim via Telegram
func GenerateQRCodeBytes(qrisCode string) ([]byte, error) {
	return qrcode.Encode(qrisCode, qrcode.Medium, 256)
}

// CRC16/CCITT-FALSE
func computeCRC16(data string) string {
	crc := 0xFFFF
	for _, c := range data {
		crc ^= int(c) << 8
		for i := 0; i < 8; i++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	crc &= 0xFFFF
	return fmt.Sprintf("%04X", crc)
}