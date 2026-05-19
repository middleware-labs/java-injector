package otelinject

import "testing"

func TestDropinValidate(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		ldPreload   string
		wantErr     bool
	}{
		{"valid name", "flask-app", DefaultLibOtelInjectorPath, false},
		{"valid with dots", "my.service", DefaultLibOtelInjectorPath, false},

		// Blocked prefixes
		{"user@ blocked", "user@1000", DefaultLibOtelInjectorPath, true},
		{"session- blocked", "session-42", DefaultLibOtelInjectorPath, true},
		{"init.scope blocked", "init.scope", DefaultLibOtelInjectorPath, true},
		{"dbus blocked", "dbus-broker", DefaultLibOtelInjectorPath, true},

		// Forbidden characters
		{"newline in name", "flask\napp", DefaultLibOtelInjectorPath, true},
		{"carriage return in name", "flask\rapp", DefaultLibOtelInjectorPath, true},
		{"quote in name", `flask"app`, DefaultLibOtelInjectorPath, true},
		{"newline in ldpreload", "flask-app", "/lib/evil\n.so", true},
		{"quote in ldpreload", "flask-app", `/lib/evil".so`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &SystemdDropin{
				ServiceName: tt.serviceName,
				LdPreload:   tt.ldPreload,
			}
			err := d.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewSystemdDropin(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantService string
	}{
		{"strips .service suffix", "flask-app.service", "flask-app"},
		{"no suffix unchanged", "flask-app", "flask-app"},
		{"double suffix stripped once", "flask-app.service.service", "flask-app.service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewSystemdDropin(tt.input)
			if err != nil {
				t.Fatalf("NewSystemdDropin(%q) error = %v", tt.input, err)
			}
			if d.ServiceName != tt.wantService {
				t.Errorf("ServiceName = %q, want %q", d.ServiceName, tt.wantService)
			}
			if d.LdPreload != DefaultLibOtelInjectorPath {
				t.Errorf("LdPreload = %q, want %q", d.LdPreload, DefaultLibOtelInjectorPath)
			}
		})
	}
}

func TestShellescape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean string", "flask-app", "flask-app"},
		{"backslash escaped", `a\b`, `a\\b`},
		{"quote escaped", `a"b`, `a\"b`},
		{"both escaped", `a\"b`, `a\\\"b`},
		{"empty", "", ""},
		{"path with slashes", "/usr/lib/libotelinject.so", "/usr/lib/libotelinject.so"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellescape(tt.input)
			if got != tt.want {
				t.Errorf("shellescape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
