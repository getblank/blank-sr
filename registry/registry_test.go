package registry

import "testing"

func TestRegister(t *testing.T) {
	s := Service{"type1", "addr1", "id1"}
	services.register(s)
	if len(services.services["type1"]) != 1 {
		t.Fatal("Service was not registered")
	}
	s = Service{"type1", "addr2", "id2"}
	services.register(s)
	if len(services.services["type1"]) != 2 {
		t.Fatal("Service was not registered")
	}
	s = Service{"type3", "addr3", "id3"}
	services.register(s)
	if len(services.services["type3"]) != 1 {
		t.Fatal("Service was not registered")
	}

	Unregister("id1")
	if len(services.services["type1"]) != 1 {
		t.Fatal("Service was not unregistered", len(services.services["type1"]))
	}
	if services.services["type1"][0].connID != "id2" {
		t.Fatal("Invalid service was unregistered")
	}
}
