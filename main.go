package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-rod/rod"
)

const (
	HOST               = "example.com"
	API_QUOTA          = 2
	API_QUOTA_PEROID   = 24
	API_QUOTA_DURATION = time.Hour
	BAN_MASSAGE        = "You have been banned for 24 hours. Better Luck Next Time!!!!\n"
)

var (
	//go:embed html/docs.html
	docsHtml []byte
	//go:embed extractor.js
	jsCode string
)

type ExtractedElement []struct {
	ImageURL string `json:"imageUrl"`
	VideoURL string `json:"videoUrl"`
	Time     string `json:"time"`
	Desc     string `json:"desc"`
}

type Server struct {
	port   string
	Access map[string][]time.Time
	mutex  sync.Mutex
}

type BanManager struct {
	mu     sync.RWMutex
	banned map[string]bool
}

func NewBanManager() *BanManager {
	return &BanManager{banned: make(map[string]bool)}
}

func (bm *BanManager) Ban(ip string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.banned[ip] = true
	log.Printf("IP Banned: %s", ip)
}

func (bm *BanManager) IsBanned(ip string) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.banned[ip]
}

func extractIP(remoteAddr string) string {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr // Fallback if format differs
	}
	return ip
}

func (s *Server) start() {
	banManager := NewBanManager()
	mux := http.NewServeMux()
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	mux.HandleFunc("/.well-known/appspecific/com.chrome.devtools.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Remote Address: %s\n", r.RemoteAddr)
		fmt.Printf("Protocol: %s\n", r.Proto)
		fmt.Printf("Method: %s\n", r.Method)
		fmt.Printf("URL: %s\n", r.URL.String())
		clientIP := extractIP(r.RemoteAddr)
		banManager.Ban(clientIP)
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte(BAN_MASSAGE))
	})
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("docs access from: %s\n", r.RemoteAddr)
		parsedTemplate, err := template.New("doc").Parse(string(docsHtml))
		if err != nil {
			log.Printf("Error parsing template: %s", err)
			w.WriteHeader(http.StatusTeapot)
			return
		}
		if err := parsedTemplate.Execute(w, struct{ Host string }{Host: HOST}); err != nil {
			log.Printf("Error executing template: %s", err)
			w.WriteHeader(http.StatusTeapot)
			return
		}
	})
	mux.HandleFunc("/api/{type}/{un}", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("api access from: %s\n", r.RemoteAddr)
		tp := r.PathValue("type")
		switch tp {
		case "c":
			break
		case "user":
			break
		default:
			fmt.Printf("failed api access from: %s\n", r.RemoteAddr)
			clientIP := extractIP(r.RemoteAddr)
			banManager.Ban(clientIP)
			return
		}
		un := r.PathValue("un")
		if un == "" {
			fmt.Printf("failed api access from: %s\n", r.RemoteAddr)
			clientIP := extractIP(r.RemoteAddr)
			banManager.Ban(clientIP)
			return
		}
		s.mutex.Lock()
		if val, ok := s.Access[extractIP(r.RemoteAddr)]; ok {
			lenVal := len(val)
			if lenVal == API_QUOTA {
				now := time.Now()
				if now.Sub(val[0]) > API_QUOTA_DURATION*API_QUOTA_PEROID {
					s.Access[extractIP(r.RemoteAddr)] = s.Access[extractIP(r.RemoteAddr)][1:]
				} else {
					w.WriteHeader(http.StatusTeapot)
					w.Write([]byte(fmt.Sprintf("API daily quota met, %v", time.Until(val[0].Add(API_QUOTA_DURATION*API_QUOTA_PEROID)))))
					s.mutex.Unlock()
					return
				}
			}
		}
		s.Access[extractIP(r.RemoteAddr)] = append(s.Access[extractIP(r.RemoteAddr)], time.Now())
		s.mutex.Unlock()
		elements, err := rumbleGet(fmt.Sprintf("https://rumble.com/%s/%s", tp, un))
		if err != nil {
			log.Printf("Failed to get rumble: %v\n", err)
		}
		output, err := json.MarshalIndent(elements, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal json: %v\n", err)
		}
		if _, err := w.Write(output); err != nil {
			log.Printf("Failed to write response: %v\n", err)
		}
	})

	server := &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
		ConnState: func(conn net.Conn, state http.ConnState) {
			clientIP := extractIP(conn.RemoteAddr().String())
			if banManager.IsBanned(clientIP) {
				log.Printf("TCP connection rejected from banned IP: %s", clientIP)
				conn.Close()
			}
		},
	}
	log.Printf("Server @ http://127.0.0.1:8080/docs")
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Server error: %v", err)
	}
}

func rumbleGet(url string) (ExtractedElement, error) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()
	page := browser.MustPage(url).MustWaitStable()
	res := page.MustEval(jsCode)
	var elements ExtractedElement
	err := res.Unmarshal(&elements)
	if err != nil {
		log.Fatalf("Failed to parse elements: %v", err)
	}
	return elements, nil
}

func main() {
	s := &Server{
		Access: make(map[string][]time.Time),
	}
	flag.StringVar(&s.port, "port", "8080", "port to serve on")
	flag.Parse()
	s.start()
}
