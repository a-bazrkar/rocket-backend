package recovery

import (
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/a-bazrkar/rocket-backend/internal/errs"
	"github.com/a-bazrkar/rocket-backend/internal/packages/errcore"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	errs.MustLoad()
	os.Exit(m.Run())
}

func TestGinPanicReturnsGenericError(t *testing.T) {
	var buf strings.Builder
	log := zerolog.New(&buf)
	r := gin.New()
	r.Use(Gin(log))
	r.GET("/boom", func(c *gin.Context) {
		ctx := errcore.WithRequestID(c.Request.Context(), "req-panic")
		c.Request = c.Request.WithContext(ctx)
		panic("panic-token-secret")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "panic-token-secret") || strings.Contains(w.Body.String(), "goroutine") {
		t.Fatalf("body leaked panic: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code":"internal.panic_recovered"`) {
		t.Fatalf("body = %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "An unexpected error occurred") {
		t.Fatalf("body = %s", w.Body.String())
	}
	line := buf.String()
	for _, part := range []string{"panic-token-secret", `"stack"`, "req-panic", `"method":"GET"`, `"path":"/boom"`} {
		if !strings.Contains(line, part) {
			t.Fatalf("log %q missing %s", line, part)
		}
	}
	if strings.Contains(line, "[Recovery]") {
		t.Fatalf("gin recovery logger duplicated the panic: %s", line)
	}
	if strings.Count(line, `"error_code":"internal.panic_recovered"`) != 1 {
		t.Fatalf("panic logged more than once: %s", line)
	}
}

func TestGinPanicIncludesStackWhenDebug(t *testing.T) {
	errcore.SetDebug(true)
	t.Cleanup(func() { errcore.SetDebug(false) })
	r := gin.New()
	r.Use(Gin(zerolog.Nop()))
	r.GET("/boom", func(c *gin.Context) {
		panic("panic-token-secret")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	raw := w.Body.String()
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d body=%s", w.Code, raw)
	}
	if !strings.Contains(raw, `"code":"internal.panic_recovered"`) || !strings.Contains(raw, `"stack"`) {
		t.Fatalf("body = %s", raw)
	}
	if strings.Contains(raw, "panic-token-secret") {
		t.Fatalf("body leaked panic value: %s", raw)
	}
}

func TestGinPanicKeepsWrittenBody(t *testing.T) {
	r := gin.New()
	r.Use(Gin(zerolog.Nop()))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "partial")
		panic("panic-after-write")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK || w.Body.String() != "partial" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "panic-after-write") {
		t.Fatal(w.Body.String())
	}
}

func TestGoPanicClosesWithoutLeaking(t *testing.T) {
	server, client := wsPair(t)
	var buf strings.Builder
	log := zerolog.New(&buf)
	Go(log, server, ConnContext{RequestID: "req-9", ConnectionID: "conn-1", UserID: 4}, func() {
		panic("panic-token-secret")
	})

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := client.ReadMessage()
	if err == nil {
		t.Fatalf("received data frame %s", data)
	}
	var closed *websocket.CloseError
	if !stderrors.As(err, &closed) {
		t.Fatalf("err = %v", err)
	}
	if closed.Code != 4000 {
		t.Fatalf("close code = %d text=%q", closed.Code, closed.Text)
	}
	if strings.Contains(closed.Text, "panic-token-secret") || strings.Contains(closed.Text, "goroutine") {
		t.Fatalf("close text leaked panic: %q", closed.Text)
	}
	if closed.Text != "An unexpected error occurred" {
		t.Fatalf("close text = %q", closed.Text)
	}

	line := buf.String()
	for _, part := range []string{"panic-token-secret", `"stack"`, "conn-1", "req-9", `"user_id":4`, `"transport":"websocket"`} {
		if !strings.Contains(line, part) {
			t.Fatalf("log %q missing %s", line, part)
		}
	}
}

func TestGoNilConnectionLogsPanic(t *testing.T) {
	logged := make(chan struct{})
	writer := &notifyWriter{ch: logged}
	log := zerolog.New(writer)
	Go(log, nil, ConnContext{ConnectionID: "solo"}, func() {
		panic("nil-conn-panic")
	})
	select {
	case <-logged:
	case <-time.After(2 * time.Second):
		t.Fatal("panic was not logged")
	}
	if !strings.Contains(writer.String(), "nil-conn-panic") {
		t.Fatalf("log = %s", writer.String())
	}
}

func TestGoPanicCloseRacesWithDataWriter(t *testing.T) {
	server, client := wsPair(t)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = server.WriteMessage(websocket.TextMessage, []byte("tick"))
		}
	}()

	Go(zerolog.Nop(), server, ConnContext{ConnectionID: "race"}, func() {
		panic("race-panic-token")
	})

	deadline := time.Now().Add(3 * time.Second)
	sawClose := false
	for time.Now().Before(deadline) {
		_ = client.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, data, err := client.ReadMessage()
		if err == nil {
			if strings.Contains(string(data), "race-panic-token") {
				t.Fatalf("data leaked panic: %s", data)
			}
			continue
		}
		var closed *websocket.CloseError
		if stderrors.As(err, &closed) && closed.Code == 4000 {
			if strings.Contains(closed.Text, "race-panic-token") {
				t.Fatalf("close leaked panic: %q", closed.Text)
			}
			sawClose = true
			break
		}
	}
	close(stop)
	wg.Wait()
	if !sawClose {
		t.Fatal("panic did not close the connection")
	}
}

type notifyWriter struct {
	mu  sync.Mutex
	buf strings.Builder
	ch  chan struct{}
}

func (w *notifyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buf.Write(p)
	select {
	case <-w.ch:
	default:
		close(w.ch)
	}
	return n, err
}

func (w *notifyWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func wsPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	accepted := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		accepted <- conn
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	server := <-accepted
	t.Cleanup(func() { _ = server.Close() })
	return server, client
}
