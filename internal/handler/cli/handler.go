package cli

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/putteror/thai-id-reader/internal/service"
)

type CLIHandler struct {
	idService service.IDCardService
}

func NewCLIHandler(idService service.IDCardService) *CLIHandler {
	return &CLIHandler{idService: idService}
}

func (h *CLIHandler) Run() {
	fmt.Println("Reading Thai ID Card (including photo)...")
	card, err := h.idService.ReadCardData()
	if err != nil {
		log.Fatalf("Error reading card: %v", err)
	}

	fmt.Println("------------- Card Information -------------")
	fmt.Printf("Citizen ID: %s\n", card.CitizenID)
	fmt.Printf("Name (TH):  %s\n", card.FullNameTH)
	fmt.Printf("Name (EN):  %s\n", card.FullNameEN)
	fmt.Printf("Birth Date: %s\n", card.BirthDate)
	fmt.Printf("Gender:     %s\n", card.Gender)

	if card.Photo != "" {
		filename := "output_photo.jpg"
		fmt.Println("Photo:      ", card.Photo)
		photoBytes, err := base64.StdEncoding.DecodeString(card.Photo)
		if err != nil {
			fmt.Printf("Error decoding photo: %v\n", err)
		} else {
			err = os.WriteFile(filename, photoBytes, 0644)
			if err != nil {
				fmt.Printf("Error saving photo: %v\n", err)
			} else {
				fmt.Printf("Photo:      Saved to %s (%d bytes decoded)\n", filename, len(photoBytes))
			}
		}
	} else {
		fmt.Println("Photo:      Not found or failed to read")
	}
	fmt.Println("--------------------------------------------")
}
