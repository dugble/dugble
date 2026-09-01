package authn

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestPrincipalContext(t *testing.T) {
	t.Parallel()

	if _, ok := PrincipalFromContext(context.Background()); ok {
		t.Fatal("PrincipalFromContext() found a missing principal")
	}

	want := Principal{Kind: PrincipalUserSession, UserID: uuid.New()}
	ctx := ContextWithPrincipal(context.Background(), want)
	got, ok := PrincipalFromContext(ctx)
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("PrincipalFromContext() = (%#v, %v), want (%#v, true)", got, ok, want)
	}
	if got := MustPrincipalFromContext(ctx); !reflect.DeepEqual(got, want) {
		t.Fatalf("MustPrincipalFromContext() = %#v, want %#v", got, want)
	}
}

func TestMustPrincipalFromContextPanicsWhenMissing(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("MustPrincipalFromContext() did not panic")
		}
	}()
	MustPrincipalFromContext(context.Background())
}
