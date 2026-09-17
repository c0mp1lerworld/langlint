package domain

import (
	"errors"
	"testing"
)

func TestNewID_RandFailure_ReturnsError(t *testing.T) {
	restore := swapRandRead(func([]byte) (int, error) {
		return 0, errors.New("rand failure")
	})
	defer restore()

	if _, err := NewID(); err == nil {
		t.Fatal("NewID() with failing randomness: want error, got nil")
	}
}

func TestMustNewID_RandFailure_Panics(t *testing.T) {
	restore := swapRandRead(func([]byte) (int, error) {
		return 0, errors.New("rand failure")
	})
	defer func() {
		restore()
		if recover() == nil {
			t.Fatal("MustNewID() with failing randomness: want panic")
		}
	}()

	MustNewID()
}

func swapRandRead(fn func([]byte) (int, error)) func() {
	previous := randRead
	randRead = fn
	return func() { randRead = previous }
}
