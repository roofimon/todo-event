package result

import (
	"errors"
	"strconv"
	"testing"

	"github.com/samber/mo"
)

func TestFlatMapTransformsSuccess(t *testing.T) {
	result := FlatMap(mo.Ok(42), func(value int) mo.Result[string] {
		return mo.Ok(strconv.Itoa(value))
	})

	if result.IsError() {
		t.Fatalf("FlatMap returned error: %v", result.Error())
	}
	if result.MustGet() != "42" {
		t.Errorf("FlatMap result = %q, want %q", result.MustGet(), "42")
	}
}

func TestFlatMapPropagatesError(t *testing.T) {
	wantErr := errors.New("input failed")
	called := false

	result := FlatMap(mo.Err[int](wantErr), func(int) mo.Result[string] {
		called = true
		return mo.Ok("unexpected")
	})

	if !errors.Is(result.Error(), wantErr) {
		t.Fatalf("FlatMap error = %v, want %v", result.Error(), wantErr)
	}
	if called {
		t.Error("FlatMap invoked mapper for an error result")
	}
}
