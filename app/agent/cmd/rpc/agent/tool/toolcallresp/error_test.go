package toolcallresp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorConstructors(t *testing.T) {
	cases := []struct {
		name   string
		build  func() (string, error)
		target error
		want   string
	}{
		{"DocumentIDRequired", NewDocumentIDRequiredErr, ErrDocumentIDRequired, "文档 ID"},
		{"UserNotIdentified", NewUserNotIdentifiedErr, ErrUserNotIdentified, "用户"},
		{"DocumentNoPermission", NewDocumentNoPermissionErr, ErrDocumentNoPermission, "权限"},
		{"DocumentNotExist", NewDocumentNotExistErr, ErrDocumentNotExist, "没有找到"},
		{"DocumentQueryFailed", func() (string, error) {
			return NewDocumentQueryFailedErr(errors.New("dial tcp"))
		}, ErrDocumentQueryFailed, "稍后再试"},
		{"DocumentListFailed", func() (string, error) {
			return NewDocumentListFailedErr(errors.New("dial tcp"))
		}, ErrDocumentListFailed, "稍后再试"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			msg, err := c.build()
			if !errors.Is(err, c.target) {
				t.Fatalf("err = %v, want %v", err, c.target)
			}
			if !strings.Contains(msg, c.want) {
				t.Fatalf("msg = %q, should contain %q", msg, c.want)
			}
		})
	}
}

func TestNaturalizeWrappedError(t *testing.T) {
	err := fmt.Errorf("wrap: %w", ErrDocumentNotExist)
	if msg := Naturalize(err); !strings.Contains(msg, "没有找到") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestNaturalizeUnknownError(t *testing.T) {
	if msg := Naturalize(errors.New("boom")); !strings.Contains(msg, "boom") {
		t.Fatalf("msg = %q", msg)
	}
}

func TestErrorHandlerMatchesNaturalize(t *testing.T) {
	if got := ErrorHandler(context.Background(), ErrUserNotIdentified); got != Naturalize(ErrUserNotIdentified) {
		t.Fatal("handler should match Naturalize")
	}
}
