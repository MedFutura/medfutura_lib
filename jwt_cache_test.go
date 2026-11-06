package medfutura_lib

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetJWTCache(t *testing.T) {
	var logins int32
	exp := time.Now().Add(24 * time.Hour).Unix()
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp)))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&logins, 1)
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintf(w, `{"status":true,"data":"h.%s.s"}`, payload)
	}))
	defer srv.Close()
	t.Setenv("AUTH_LINK", srv.URL)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tk, err := GetJWT(); err != nil || tk != "Bearer h."+payload+".s" {
				t.Errorf("token %q err %v", tk, err)
			}
		}()
	}
	wg.Wait()
	if logins != 1 {
		t.Fatalf("esperava 1 login, teve %d", logins)
	}
	if d := time.Until(jwtCache.expira); d < 23*time.Hour {
		t.Fatalf("expiração errada: %v", d)
	}

	jwtCache.expira = time.Now().Add(-time.Second)
	GetJWT()
	if logins != 2 {
		t.Fatalf("esperava renovar após expirar, logins %d", logins)
	}
}
