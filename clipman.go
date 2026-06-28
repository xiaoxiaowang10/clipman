package clipman

import (
	"bufio"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func Run() {
	LoadAll()
	startFlusher()
	go monitor()
	startHTTPServer()
	openBrowser()

	h := func() uintptr { r, _, _ := procGetModuleHandleW.Call(0); return r }()
	hInst = syscall.Handle(h)
	hwndMain = createMainWindow(hInst)

	var m struct {
		hwnd    syscall.Handle
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      POINT
	}
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func monitor() {
	for {
		time.Sleep(500 * time.Millisecond)
		text := getClipText()
		if text == "" {
			continue
		}
		if ownCopy.Swap(false) {
			lastText = text
			continue
		}
		if text == lastText {
			continue
		}
		lastText = text
		mu.Lock()
		dup := false
		for _, e := range history {
			if e.Text == text {
				dup = true
				break
			}
		}
		if dup {
			mu.Unlock()
			continue
		}
		e := Entry{Text: text, Time: time.Now().Format("2006-01-02 15:04:05")}
		history = append([]Entry{e}, history...)
		if len(history) > maxEntries {
			history = history[:maxEntries]
		}
		AppendEntry(e)
		mu.Unlock()
	}
}

func startHTTPServer() {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/api", HandleAPI)
	mux.HandleFunc("/api/export", HandleExport)
	mux.HandleFunc("/api/import", HandleImport)
	mux.HandleFunc("/api/autostart", HandleAutoStart)
	srv := &http.Server{
		Addr:         "127.0.0.1:" + httpPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	go srv.ListenAndServe()
}

func HandleAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		idsStr := r.URL.Query()["id"]
		if len(idsStr) == 0 {
			ClearAll()
			w.WriteHeader(204)
			return
		}
		var ids []int
		for _, s := range idsStr {
			id, err := strconv.Atoi(s)
			if err != nil {
				continue
			}
			ids = append(ids, id)
		}
		RemoveEntries(ids)
		w.WriteHeader(204)
		return
	}

	if r.Method == "PATCH" {
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}
		mu.Lock()
		if id >= 0 && id < len(history) {
			history[id].Pinned = !history[id].Pinned
		}
		mu.Unlock()
		w.WriteHeader(204)
		return
	}

	if r.Method == "PUT" {
		idStr := r.URL.Query().Get("id")
		text := r.URL.Query().Get("text")
		id, err := strconv.Atoi(idStr)
		if err != nil || text == "" {
			http.Error(w, "invalid id or text", 400)
			return
		}
		mu.Lock()
		if id >= 0 && id < len(history) {
			history[id].Text = text
		}
		mu.Unlock()
		w.WriteHeader(204)
		return
	}

	q := strings.ToLower(r.URL.Query().Get("q"))
	mu.Lock()
	defer mu.Unlock()
	pinned := make([]Entry, 0)
	unpinned := make([]Entry, 0)
	for _, e := range history {
		if q != "" && !TokenMatch(e.Text, e.Time, q) {
			continue
		}
		if e.Pinned {
			pinned = append(pinned, e)
		} else {
			unpinned = append(unpinned, e)
		}
	}
	res := append(pinned, unpinned...)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func TokenMatch(text, time, query string) bool {
	terms := strings.Fields(query)
	combined := strings.ToLower(text) + " " + strings.ToLower(time)
	for _, t := range terms {
		if !strings.Contains(combined, t) {
			return false
		}
	}
	return true
}

func HandleExport(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(DataFile)
	if err != nil {
		http.Error(w, "read error", 500)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=clipman.jl")
	w.Write(b)
}

func HandleImport(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	f, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file", 400)
		return
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Text == "" {
			continue
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		http.Error(w, "no valid entries", 400)
		return
	}

	ResetWriter()
	fw, err := os.Create(DataFile)
	if err != nil {
		http.Error(w, "write error", 500)
		return
	}
	defer fw.Close()
	enc := json.NewEncoder(fw)
	for i := len(entries) - 1; i >= 0; i-- {
		enc.Encode(entries[i])
	}
	LoadAll()
}

func HandleAutoStart(w http.ResponseWriter, r *http.Request) {
	SetAutoStart(r.URL.Query().Get("enable") == "true")
}

func openBrowser() {
	exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://127.0.0.1:"+httpPort).Start()
}

//go:embed web/*
var webFS embed.FS
