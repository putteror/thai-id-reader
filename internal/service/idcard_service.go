package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/putteror/thai-id-reader/internal/adapter/pcsc"
)

type ThaiIDCard struct {
	CitizenID  string
	FullNameTH string
	FullNameEN string
	BirthDate  string
	Gender     string
	Photo      string
}

type IDCardService interface {
	ReadCardData() (*ThaiIDCard, error)
}

type thaiIDService struct {
	reader pcsc.Reader
}

func NewThaiIDService(reader pcsc.Reader) IDCardService {
	return &thaiIDService{reader: reader}
}

// APDU Commands for Thai ID
var (
	cmdSelectApplet = []byte{0x00, 0xA4, 0x04, 0x00, 0x08, 0xA0, 0x00, 0x00, 0x00, 0x54, 0x48, 0x00, 0x01}
	cmdCitizenID    = []byte{0x80, 0xb0, 0x00, 0x04, 0x02, 0x00, 0x0d}
	cmdFullNameTH   = []byte{0x80, 0xb0, 0x00, 0x11, 0x02, 0x00, 0x64}
	cmdFullNameEN   = []byte{0x80, 0xb0, 0x00, 0x75, 0x02, 0x00, 0x64}
	cmdBirthDate    = []byte{0x80, 0xb0, 0x00, 0xD9, 0x02, 0x00, 0x08}
	cmdGender       = []byte{0x80, 0xb0, 0x00, 0xE1, 0x02, 0x00, 0x01}
)

func (s *thaiIDService) ReadCardData() (*ThaiIDCard, error) {
	// 1. Select Applet
	rsp, err := s.reader.Transmit(cmdSelectApplet)
	if err != nil {
		return nil, fmt.Errorf("failed to select applet: %w", err)
	}
	if !isSuccessOrHasBytes(rsp) {
		return nil, fmt.Errorf("failed to select applet: unexpected response %X", rsp)
	}

	// 2. Read basic info fields...
	citizenID, _ := s.readAndClean(cmdCitizenID)
	fullNameTH, _ := s.readAndClean(cmdFullNameTH)
	fullNameEN, _ := s.readAndClean(cmdFullNameEN)
	birthDate, _ := s.readAndClean(cmdBirthDate)
	gender, _ := s.readAndClean(cmdGender)

	// 3. Read Photo data (20 chunks of 255 bytes)
	photoBase64, err := s.readPhoto()
	if err != nil {
		fmt.Printf("Warning: failed to read photo: %v\n", err)
	}

	card := &ThaiIDCard{
		CitizenID:  citizenID,
		FullNameTH: fullNameTH,
		FullNameEN: fullNameEN,
		BirthDate:  birthDate,
		Gender:     gender,
		Photo:      photoBase64,
	}

	return card, nil
}

func (s *thaiIDService) readPhoto() (string, error) {
	var photoData []byte
	// รูปในบัตรประชาชนมักจะมีขนาดประมาณ 5,120 bytes
	// และ offset เริ่มต้นคือ 0x017B (379)
	offset := 0x017B

	// 5120 / 252 = 20.3 -> ต้องอ่านประมาณ 21 รอบ
	for i := 0; i < 21; i++ {
		p1 := byte((offset >> 8) & 0xFF)
		p2 := byte(offset & 0xFF)

		// อ่านทีละ 252 bytes (0xFC)
		readCmd := []byte{0x80, 0xB0, p1, p2, 0x02, 0x00, 0xFC}

		rsp, err := s.reader.Transmit(readCmd)
		if err != nil {
			return "", fmt.Errorf("failed chunk %d: %w", i, err)
		}

		// เช็คว่าต้องส่ง Get Response ไหม (0x61)
		if len(rsp) >= 2 && rsp[len(rsp)-2] == 0x61 {
			lenToRead := rsp[len(rsp)-1]
			getRespCmd := []byte{0x00, 0xC0, 0x00, 0x00, lenToRead}
			rsp, err = s.reader.Transmit(getRespCmd)
			if err != nil {
				return "", err
			}
		}

		// เก็บข้อมูล (ตัด 2 byte สุดท้ายที่เป็น Status Word ออก)
		if len(rsp) > 2 {
			photoData = append(photoData, rsp[:len(rsp)-2]...)
		}

		offset += 252
	}

	// --- ขั้นตอนสำคัญ: Clean JPEG Data ---
	// ค้นหาตำแหน่งเริ่มต้น FF D8 และจุดสิ้นสุด FF D9
	startIndex := bytes.Index(photoData, []byte{0xFF, 0xD8})
	endIndex := bytes.LastIndex(photoData, []byte{0xFF, 0xD9})

	if startIndex == -1 || endIndex == -1 || endIndex <= startIndex {
		// Print first 50 bytes for debugging if JPEG header not found
		if len(photoData) > 50 {
			fmt.Printf("Debug Read: %X\n", photoData[:50])
		}
		return "", fmt.Errorf("could not find valid JPEG markers in photo data")
	}

	finalPhoto := photoData[startIndex : endIndex+2]
	return base64.StdEncoding.EncodeToString(finalPhoto), nil
}

func (s *thaiIDService) readAndClean(setupCmd []byte) (string, error) {
	// 1. Setup read
	rsp, err := s.reader.Transmit(setupCmd)
	if err != nil {
		return "", fmt.Errorf("failed to setup read command: %w", err)
	}

	length := setupCmd[len(setupCmd)-1]

	// The reader might return 61 XX if data is ready, or 90 00 indicating success.
	// But usually, standard APDU requires sending `0x00 0xC0...` to read pending bytes.

	getRespCmd := []byte{0x00, 0xC0, 0x00, 0x00, length}
	dataRsp, err := s.reader.Transmit(getRespCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}

	if len(dataRsp) < 2 {
		// Fallback to initial response if it somehow holds the data
		if len(rsp) > 2 {
			data := rsp[:len(rsp)-2]
			return cleanString(tis620ToUTF8(data)), nil
		}
		return "", fmt.Errorf("read failed, bad response: %X", dataRsp)
	}

	// Strip the last 2 bytes (status word)
	data := dataRsp[:len(dataRsp)-2]

	cleaned := bytes.TrimSpace(data)
	return cleanString(tis620ToUTF8(cleaned)), nil
}

func isSuccessOrHasBytes(rsp []byte) bool {
	if len(rsp) < 2 {
		return false
	}
	sw1 := rsp[len(rsp)-2]
	sw2 := rsp[len(rsp)-1]

	// 90 00 is OK
	if sw1 == 0x90 && sw2 == 0x00 {
		return true
	}
	// 61 XX indicates more bytes are available
	if sw1 == 0x61 {
		return true
	}
	return false
}

func tis620ToUTF8(tis620Bytes []byte) string {
	var utf8Runes []rune
	for _, b := range tis620Bytes {
		if b < 0x80 {
			utf8Runes = append(utf8Runes, rune(b))
		} else if b >= 0xA1 {
			utf8Runes = append(utf8Runes, rune(b)+0x0E00-0xA0)
		}
	}
	return string(utf8Runes)
}

func cleanString(s string) string {
	// Remove redundant # delimiter in names and replace with spaces
	s = strings.ReplaceAll(s, "##", " ")
	s = strings.ReplaceAll(s, "#", " ")
	return strings.TrimSpace(s)
}
