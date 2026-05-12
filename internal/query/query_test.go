package query

import "testing"

func TestEncode_DropsNil(t *testing.T) {
	p := Params{}
	p.Set("a", "x")
	p.Set("b", nil)
	p.Set("c", "y")
	got := EncodeToString(p)
	want := "a=x&c=y"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_NilPointerDropped(t *testing.T) {
	var b *bool
	p := Params{}
	p.Set("flag", b)
	if got := EncodeToString(p); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestEncode_BoolStringified(t *testing.T) {
	tru := true
	fls := false
	p := Params{}
	p.Set("a", true)
	p.Set("b", false)
	p.Set("c", &tru)
	p.Set("d", &fls)
	want := "a=true&b=false&c=true&d=false"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_SelectorsExplodedWithoutBrackets(t *testing.T) {
	p := Params{}
	p.Set("selectors", []string{"h1", ".price", "#main"})
	want := "selectors=h1&selectors=.price&selectors=%23main"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_HeadersUseDeepObjectBrackets(t *testing.T) {
	p := Params{}
	p.Set("headers", map[string]string{
		"Cookie":     "session=abc",
		"User-Agent": "MyBot/1.0",
	})
	// Sub-keys are sorted alphabetically for determinism.
	want := "headers%5BCookie%5D=session%3Dabc&headers%5BUser-Agent%5D=MyBot%2F1.0"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_FieldsUseDeepObjectBrackets(t *testing.T) {
	p := Params{}
	p.Set("fields", map[string]string{
		"title": "Page title",
		"price": "Listed price",
	})
	want := "fields%5Bprice%5D=Listed%20price&fields%5Btitle%5D=Page%20title"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_SpacesPercentTwenty(t *testing.T) {
	p := Params{}
	p.Set("q", "hello world")
	if got := EncodeToString(p); got != "q=hello%20world" {
		t.Fatalf("got %q, want q=hello%%20world", got)
	}
}

func TestEncode_PreservesInsertionOrder(t *testing.T) {
	p := Params{}
	p.Set("z", "1")
	p.Set("a", "2")
	p.Set("m", "3")
	want := "z=1&a=2&m=3"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_SetReplacesInPlace(t *testing.T) {
	p := Params{}
	p.Set("a", "1")
	p.Set("b", "2")
	p.Set("a", "3")
	want := "a=3&b=2"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_Numerics(t *testing.T) {
	p := Params{}
	p.Set("i", 42)
	p.Set("i64", int64(99))
	p.Set("f", 3.14)
	want := "i=42&i64=99&f=3.14"
	if got := EncodeToString(p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEncode_EmptyParamsEmptyString(t *testing.T) {
	if got := EncodeToString(Params{}); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestEncode_DeepObjectSortedDeterministically(t *testing.T) {
	// Same map twice — both calls produce the same output.
	m := map[string]string{"c": "3", "a": "1", "b": "2"}
	p1 := Params{}
	p1.Set("h", m)
	p2 := Params{}
	p2.Set("h", m)
	if EncodeToString(p1) != EncodeToString(p2) {
		t.Fatal("encoding map values is not deterministic")
	}
	if got := EncodeToString(p1); got != "h%5Ba%5D=1&h%5Bb%5D=2&h%5Bc%5D=3" {
		t.Fatalf("got %q", got)
	}
}

func TestEncode_PanicsOnUnsupportedType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unsupported type")
		}
	}()
	p := Params{}
	p.Set("bad", struct{}{})
	_ = EncodeToString(p)
}
