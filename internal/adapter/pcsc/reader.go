package pcsc

import (
	"fmt"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

type Reader interface {
	Connect() error
	Disconnect() error
	Transmit(cmd []byte) ([]byte, error)
}

type pcscReader struct {
	context *scard.Context
	card    *scard.Card
}

func NewReader() Reader {
	return &pcscReader{}
}

func (r *pcscReader) Connect() error {
	var err error
	r.context, err = scard.EstablishContext()
	if err != nil {
		return fmt.Errorf("failed to establish context: %w", err)
	}

	readers, err := r.context.ListReaders()
	if err != nil {
		r.context.Release()
		return fmt.Errorf("failed to list readers: %w", err)
	}

	if len(readers) == 0 {
		r.context.Release()
		return fmt.Errorf("no smart card readers found")
	}

	// Use the first available reader
	readerName := readers[0]

	// Initial attempt to connect
	r.card, err = r.context.Connect(readerName, scard.ShareShared, scard.ProtocolT0)
	// r.card, err = r.context.Connect(readerName, scard.ShareShared, scard.ProtocolAny)

	if err != nil {
		// If the card is unresponsive, try to force a reset
		if strings.Contains(err.Error(), "unresponsive") {
			fmt.Println("🔄 บัตรไม่ตอบสนอง กำลังพยายาม Reset บัตร...")

			// Try to get a direct handle to the reader
			directCard, e := r.context.Connect(readerName, scard.ShareDirect, 0)
			if e == nil {
				// Force reset via Disconnect
				directCard.Disconnect(scard.ResetCard)
			}

			// Wait for reader to stabilize
			time.Sleep(500 * time.Millisecond)

			// Retry normal connection
			r.card, err = r.context.Connect(readerName, scard.ShareShared, scard.ProtocolAny)
			if err != nil {
				r.context.Release()
				return fmt.Errorf("failed to recover unresponsive card: %w", err)
			}
		} else {
			r.context.Release()
			return fmt.Errorf("failed to connect to reader %s: %w", readerName, err)
		}
	}

	return nil
}

func (r *pcscReader) Disconnect() error {
	if r.card != nil {
		r.card.Disconnect(scard.LeaveCard)
	}
	if r.context != nil {
		r.context.Release()
	}
	return nil
}

func (r *pcscReader) Transmit(cmd []byte) ([]byte, error) {
	if r.card == nil {
		return nil, fmt.Errorf("card is not connected")
	}
	rsp, err := r.card.Transmit(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to transmit APDU: %w", err)
	}
	return rsp, nil
}
