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

func TestFlatMap3ComposesStages(t *testing.T) {
	result := FlatMap3(
		mo.Ok("42"),
		func(value string) mo.Result[int] {
			parsed, err := strconv.Atoi(value)
			return mo.TupleToResult(parsed, err)
		},
		func(value int) mo.Result[bool] {
			return mo.Ok(value == 42)
		},
	)

	if result.IsError() {
		t.Fatalf("FlatMap3 returned error: %v", result.Error())
	}
	if !result.MustGet() {
		t.Error("FlatMap3 result = false, want true")
	}
}

func TestFlatMap3ShortCircuits(t *testing.T) {
	wantErr := errors.New("first stage failed")
	secondCalled := false

	result := FlatMap3(
		mo.Ok("input"),
		func(string) mo.Result[int] {
			return mo.Err[int](wantErr)
		},
		func(int) mo.Result[bool] {
			secondCalled = true
			return mo.Ok(true)
		},
	)

	if !errors.Is(result.Error(), wantErr) {
		t.Fatalf("FlatMap3 error = %v, want %v", result.Error(), wantErr)
	}
	if secondCalled {
		t.Error("FlatMap3 invoked the second mapper after the first failed")
	}
}
