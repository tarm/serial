package serial

import (
	"os"
	"testing"
)

func TestListRX(t *testing.T) {
	port0 := os.Getenv("PORT0")
	port1 := os.Getenv("PORT1")

	names, err := ListRX()
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range names {
		t.Log("got", name)

		switch {
		case name == port0:
			port0 = ""
		case name == port1:
			port1 = ""
		}
	}

	if port0 != "" {
		t.Errorf("PORT0=%q not found", port0)
	}
	if port1 != "" {
		t.Errorf("PORT1=%q not found", port1)
	}
}
