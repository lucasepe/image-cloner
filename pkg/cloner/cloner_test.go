package cloner_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/lucasepe/image-cloner/pkg/cloner"
)

const (
	targetUser = "testuser"
	targetPass = "testpass"

	srcImage = "nginx:1.21.4-alpine"
)

// TestCloner it's just a way to show how
// to use the `Cloner`. It's not a proper test.
func TestCloner(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		//switch pt := req.URL.Path; pt {
		//default:
		//	fmt.Println("==> ", pt)
		//}
		w.WriteHeader(http.StatusOK)
	}))
	// Close the server when test finishes
	defer server.Close()

	targetRegistry := strings.TrimPrefix(server.URL, "http://")

	ic := cloner.New(targetRegistry,
		cloner.Credentials{
			Username: targetUser,
			Password: targetPass,
		})

	act, err := ic.CloneEventually(srcImage)
	if err != nil {
		panic(err)
	}

	exp := fmt.Sprintf("%s/%s", targetRegistry, srcImage)
	equals(t, exp, act)
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("%s:%d:\n\texp: %#v\n\tgot: %#v\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
