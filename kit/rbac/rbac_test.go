package rbac

import "testing"

func TestAllowsExactMatch(t *testing.T) {
	policies := []Policy{{Resource: "tasks", Action: "write"}}
	if !Allows(policies, "tasks", "write") {
		t.Fatal("expected allow for exact match")
	}
}

func TestWriteImpliesRead(t *testing.T) {
	policies := []Policy{{Resource: "tasks", Action: "write"}}
	if !Allows(policies, "tasks", "read") {
		t.Fatal("write policy should imply read")
	}
}

func TestReadDoesNotImplyWrite(t *testing.T) {
	policies := []Policy{{Resource: "tasks", Action: "read"}}
	if Allows(policies, "tasks", "write") {
		t.Fatal("read policy must not grant write")
	}
}

func TestMissingResourceDenies(t *testing.T) {
	policies := []Policy{{Resource: "tasks", Action: "write"}}
	if Allows(policies, "deliveries", "write") {
		t.Fatal("must deny when resource not in policy set")
	}
}

func TestEmptyPoliciesDeny(t *testing.T) {
	if Allows(nil, "tasks", "read") {
		t.Fatal("empty policy set must deny")
	}
}

func TestValidKnown(t *testing.T) {
	if !Valid("tasks", "write") {
		t.Fatal("tasks:write should be a valid resource/action")
	}
}

func TestValidUnknown(t *testing.T) {
	if Valid("unknown", "read") {
		t.Fatal("unknown resource must not validate")
	}
	if Valid("tasks", "delete") {
		t.Fatal("unknown action must not validate")
	}
}
