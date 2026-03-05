package wecom

import (
	"testing"
)

func TestType(t *testing.T) {
	if Type != "wecom" {
		t.Fatalf("Type = %q, want wecom", Type)
	}
}

func TestAdapterType(t *testing.T) {
	a := NewWeComAdapter(nil)
	if a.Type() != Type {
		t.Fatalf("Type() = %q", a.Type())
	}
}

func TestAdapterDescriptor(t *testing.T) {
	a := NewWeComAdapter(nil)
	d := a.Descriptor()

	if d.Type != Type {
		t.Fatalf("descriptor type = %q", d.Type)
	}
	if d.DisplayName == "" {
		t.Fatal("display name should not be empty")
	}
	if !d.Capabilities.Text {
		t.Fatal("should support text")
	}
	if d.ConfigSchema.Version == 0 {
		t.Fatal("config schema version should not be 0")
	}
	// Check required fields exist.
	requiredFields := []string{"corpId", "agentId", "secret", "token", "encodingAESKey"}
	for _, f := range requiredFields {
		field, ok := d.ConfigSchema.Fields[f]
		if !ok {
			t.Fatalf("missing config field %q", f)
		}
		if !field.Required {
			t.Fatalf("field %q should be required", f)
		}
	}
}

func TestNormalizeConfig(t *testing.T) {
	a := NewWeComAdapter(nil)
	raw := map[string]any{
		"corpId":        "ww123",
		"agentId":       "100",
		"secret":        "sec",
		"token":         "tok",
		"encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
	result, err := a.NormalizeConfig(raw)
	if err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if result["corpId"] != "ww123" {
		t.Fatalf("corpId = %v", result["corpId"])
	}
}
