package clipman

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Entry struct {
	Text   string `json:"text"`
	Time   string `json:"time"`
	Pinned bool   `json:"pinned,omitempty"`
}

var (
	history    []Entry
	lastText   string
	DataFile   string
	ownCopy    atomic.Bool
	mu         sync.Mutex
	httpPort = "16273"

	dataBuf     *bufio.Writer
	dataFile    *os.File
	fileMu      sync.Mutex
	currentFile string
)

const maxEntries = 1000

func AppendEntry(e Entry) {
	fileMu.Lock()
	defer fileMu.Unlock()

	if dataBuf == nil || DataFile != currentFile {
		if dataFile != nil {
			dataBuf.Flush()
			dataFile.Close()
		}
		f, err := os.OpenFile(DataFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		dataFile = f
		dataBuf = bufio.NewWriter(f)
		currentFile = DataFile
	}
	json.NewEncoder(dataBuf).Encode(e)
	dataBuf.Flush()
}

func FlushData() {
	fileMu.Lock()
	defer fileMu.Unlock()
	if dataBuf != nil {
		dataBuf.Flush()
	}
}

func LoadAll() {
	mu.Lock()
	defer mu.Unlock()
	if DataFile == "" {
		DataFile = filepath.Join(ExeDir(), "clipman.jl")
	}
	f, err := os.Open(DataFile)
	if err != nil {
		history = nil
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var all []Entry
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		all = append(all, e)
	}

	history = make([]Entry, len(all))
	for i, e := range all {
		history[len(all)-1-i] = e
	}
}

func ExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func SetAutoStart(enable bool) {
	k := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if enable {
		exe, _ := os.Executable()
		exec.Command("reg", "add", k, "/v", "Clipman", "/t", "REG_SZ", "/d", exe, "/f").Run()
	} else {
		exec.Command("reg", "delete", k, "/v", "Clipman", "/f").Run()
	}
}

func ResetWriter() {
	fileMu.Lock()
	defer fileMu.Unlock()
	if dataFile != nil {
		dataBuf.Flush()
		dataFile.Close()
		dataFile = nil
		dataBuf = nil
		currentFile = ""
	}
}

func ClearAll() {
	mu.Lock()
	defer mu.Unlock()
	history = nil

	fileMu.Lock()
	defer fileMu.Unlock()
	if dataFile != nil {
		dataBuf.Flush()
		dataFile.Close()
		dataFile = nil
		dataBuf = nil
		currentFile = ""
	}
	os.Create(DataFile)
}

func RemoveEntries(ids []int) {
	mu.Lock()
	defer mu.Unlock()

	remove := make(map[int]bool)
	for _, id := range ids {
		if id >= 0 && id < len(history) {
			remove[id] = true
		}
	}
	if len(remove) == 0 {
		return
	}

	var keep []Entry
	for i, e := range history {
		if !remove[i] {
			keep = append(keep, e)
		}
	}
	history = keep

	fileMu.Lock()
	defer fileMu.Unlock()
	if dataFile != nil {
		dataBuf.Flush()
		dataFile.Close()
		dataFile = nil
		dataBuf = nil
		currentFile = ""
	}
	f, err := os.Create(DataFile)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for i := len(history) - 1; i >= 0; i-- {
		enc.Encode(history[i])
	}
}

func startFlusher() {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			FlushData()
		}
	}()
}
