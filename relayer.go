package wsrelayer

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cespare/xxhash"
	"github.com/gorilla/websocket"
	"github.com/valyala/fastjson"
)

var (
	requestCache = NewRequestCache()
)

type WSRelayer struct {
	Port           int
	RequestTimeout time.Duration
	conn           *websocket.Conn
}

func (r *WSRelayer) ConnectAndServe(endpoint string) error {
	// connect to wss endpoint
	c, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		log.Fatal("<== dial:", err)
		return err
	}

	r.conn = c
	log.Printf("==> Relay %s", endpoint)
	go r.serveHTTP()

	// process ws messages
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			err := r.process()
			if err != nil {
				return
			}
		}
	}()

	// keepalvie timer
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// handle ctrl-c
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-done:
			return nil
		case <-ticker.C:
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				log.Println("write:", err)
				return err
			}
		case <-interrupt:
			log.Println("ctrl-c interrupt")
			r.Stop()
			return nil
		}
	}
}

func (r *WSRelayer) Stop() {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := r.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
	time.Sleep(time.Second)
}

func (r *WSRelayer) serveHTTP() {
	http.HandleFunc("/", r.relayHandler)
	log.Printf("==> Relaying at 0.0.0.0:%d\n", r.Port)
	if http.ListenAndServe(fmt.Sprintf(":%d", r.Port), nil) != nil {
		panic("<== port is not available")
	}
}

func (r *WSRelayer) relayHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" || req.Method != "POST" {
		http.NotFound(w, req)
		return
	}

	value, originalId, err := parseJSONRequest(req)
	if err != nil || originalId == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "Invalid jsonrpc request")
		return
	}

	id := genUniqueId(req)
	ch := make(chan []byte)
	item := CacheItem{
		Id:         id,
		OriginalId: originalId,
		Chan:       ch,
	}
	requestCache.Set(id, item, time.Now().Add(r.RequestTimeout))
	log.Printf("==> relay: %s", value.String())
	value.Set("id", fastjson.MustParse(id))

	err = r.conn.WriteMessage(websocket.TextMessage, []byte(value.String()))
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, err.Error())
		return
	}

	select {
	case <-time.After(r.RequestTimeout * time.Second):
		log.Println("<== wait response timeout")
		w.WriteHeader(http.StatusServiceUnavailable)
		close(ch)
	case resp := <-ch:
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(resp)
	}
}

func (r *WSRelayer) process() error {
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		log.Println("<== read:", err)
		return err
	}

	log.Printf("<== recv: %s\n", message)
	value, err := parseJSON(message)
	if err != nil {
		log.Println("<== parse json error:", err)
		return err
	}

	id := extractId(value)
	if item := requestCache.Get(id); item != nil {
		value.Set("id", fastjson.MustParse(item.OriginalId))
		item.Chan <- []byte(value.String())
	}
	return nil
}

func parseJSONRequest(req *http.Request) (*fastjson.Value, string, error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, "", err
	}
	value, err := parseJSON(body)
	originalId := extractId(value)
	return value, originalId, err
}

func parseJSON(data []byte) (*fastjson.Value, error) {
	var p fastjson.Parser
	return p.ParseBytes(data)
}

func extractIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func extractId(value *fastjson.Value) string {
	if id := value.GetInt64("id"); id != 0 {
		return strconv.FormatInt(id, 10)
	}
	return string(value.GetStringBytes("id"))
}

func genUniqueId(r *http.Request) string {
	ip := extractIP(r)
	id := fmt.Sprintf("%d-%s", time.Now().UnixNano(), ip)
	return fmt.Sprintf("%d", xxhash.Sum64String(id))
}
