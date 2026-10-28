package toolkit

import (
	"testing"
)

func TestPtr(t *testing.T) {
	// 普通值测试
	val := 123
	ptr := Ptr(val)
	if ptr == nil || *ptr != 123 {
		t.Fatalf("Ptr(123) = %v, want 123", ptr)
	}

	// 字符串测试
	str := "hello"
	sPtr := Ptr(str)
	if sPtr == nil || *sPtr != "hello" {
		t.Fatalf("Ptr(\"hello\") = %v, want hello", sPtr)
	}

	// nil 指针传入测试
	var nilPtr *int
	res := Ptr(nilPtr)
	if res != nil {
		t.Fatalf("Ptr(nil) should return nil, got %v", res)
	}
}

func TestVal(t *testing.T) {
	val := 42
	if got := Val(&val); got != 42 {
		t.Fatalf("Val(&42) = %v, want 42", got)
	}

	var nilInt *int
	if got := Val(nilInt); got != 0 {
		t.Fatalf("Val(nil) = %v, want 0", got)
	}

	var nilStr *string
	if got := Val(nilStr); got != "" {
		t.Fatalf("Val(nil) = %q, want \"\"", got)
	}
}

func TestContains(t *testing.T) {
	intSlice := []int{1, 2, 3, 4, 5}
	if !Contains(intSlice, 3) {
		t.Errorf("Contains(intSlice, 3) = false, want true")
	}
	if Contains(intSlice, 6) {
		t.Errorf("Contains(intSlice, 6) = true, want false")
	}

	strSlice := []string{"apple", "banana", "cherry"}
	if !Contains(strSlice, "banana") {
		t.Errorf("Contains(strSlice, \"banana\") = false, want true")
	}
	if Contains(strSlice, "orange") {
		t.Errorf("Contains(strSlice, \"orange\") = true, want false")
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		input        string
		defaultValue bool
		want         bool
	}{
		{"true", false, true},
		{"True", false, true},
		{"TRUE", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"YES", false, true},
		{"on", false, true},
		{"ON", false, true},
		{"enabled", false, true},
		{"false", true, false},
		{"False", true, false},
		{"0", true, false},
		{"no", true, false},
		{"off", true, false},
		{"disabled", true, false},
		{"", false, false},
		{"", true, true},
		{"invalid", false, false},
		{"invalid", true, true},
		{" 1 ", false, true},
		{" false ", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ToBool(tt.input, tt.defaultValue); got != tt.want {
				t.Errorf("ToBool(%q, %v) = %v, want %v", tt.input, tt.defaultValue, got, tt.want)
			}
		})
	}
}
