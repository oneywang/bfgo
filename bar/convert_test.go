package bar

import (
	"log"
	"testing"
)

func TestNullString(t *testing.T) {
	if false {
		t.Errorf("expecting period")
	}
	log.Print("empty")
}
