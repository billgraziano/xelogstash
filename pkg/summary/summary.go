package summary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	humanize "github.com/dustin/go-humanize"
	"github.com/pkg/errors"
)

type stats struct {
	Event  string
	Count  int64
	Size   int64
	Max    int64
	Sample string
}

var events map[string]*stats
var mux sync.RWMutex

func init() {
	mux.Lock()
	defer mux.Unlock()

	events = make(map[string]*stats)
}

// Add to the count for an event
func Add(e string, body *string) {
	e = strings.ToLower(e)
	size := int64(len(*body))
	mux.Lock()
	defer mux.Unlock()

	st, ok := events[e]
	if ok {
		st.Count++
		st.Size += size
		if size > st.Max {
			st.Max = size
		}
		return
	}
	newStats := stats{Event: e, Count: 1, Size: size, Max: size, Sample: *body}
	events[e] = &newStats
}

// PrintSummary outputs the results to stdout
func PrintSummary() {
	mux.RLock()
	defer mux.RUnlock()

	if len(events) == 0 {
		return
	}

	results := make([]stats, 0)

	var totals stats
	for _, v := range events {
		results = append(results, *v)
		totals.Count += v.Count
		totals.Size += v.Size
		if v.Max > totals.Max {
			totals.Max = v.Max
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[j].Count < results[i].Count })

	fmt.Printf("------------------------------------------------------------------------------------\r\n")
	fmt.Printf("| %-40s| %11s | %6s | %6s | %6s |\r\n", "EVENT", "COUNT", "SIZE", "AVG", "MAX")
	fmt.Printf("------------------------------------------------------------------------------------\r\n")
	for _, v := range results {
		//fmt.Println(v.Event, v.Count, v.Size)
		fmt.Printf("| %-40s| %11s | %6s | %6s | %6s |\r\n",
			v.Event,
			humanize.Comma(v.Count),
			humanize.Bytes(uint64(v.Size)),
			humanize.Bytes(uint64(v.Size/v.Count)),
			humanize.Bytes(uint64(v.Max)),
		)
	}

	fmt.Printf("------------------------------------------------------------------------------------\r\n")

	fmt.Printf("| %-40s| %11s | %6s | %6s | %6s |\r\n",
		"TOTAL",
		humanize.Comma(totals.Count),
		humanize.Bytes(uint64(totals.Size)),
		humanize.Bytes(uint64(totals.Size/totals.Count)),
		humanize.Bytes(uint64(totals.Max)),
	)
	fmt.Printf("------------------------------------------------------------------------------------\r\n")
}

// PrintSamples outputs a sample of each type of samples.txt
func PrintSamples() error {
	var err error
	results := make([]stats, 0)

	for _, v := range events {
		results = append(results, *v)
	}

	sort.SliceStable(results, func(i, j int) bool { return results[j].Event > results[i].Event })

	// Get EXE directory
	executable, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "os.executable")
	}
	exeDir := filepath.Dir(executable)
	fileName := filepath.Clean(filepath.Join(exeDir, "samples.xe.json"))

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return errors.Wrap(err, "os.openfile")
	}
	defer file.Close() //#nosec G307

	for _, v := range results {
		//json, err := json.Unmarshal(v.Sample)
		// z := bytes.NewBufferString(v.Sample)
		var out bytes.Buffer
		err := json.Indent(&out, []byte(v.Sample), "", "  ")
		if err != nil {
			return errors.Wrap(err, "json.indent")
		}
		if _, err = file.Write([]byte("------------------------------------------------------\r\n")); err != nil {
			return errors.Wrap(err, "file.write")
		}
		if _, err = file.Write([]byte(fmt.Sprintf("| EVENT: %-44s|\r\n", v.Event))); err != nil {
			return errors.Wrap(err, "file.write")
		}
		if _, err = file.Write([]byte("------------------------------------------------------\r\n")); err != nil {
			return errors.Wrap(err, "file.write")
		}
		if _, err = file.Write([]byte(fmt.Sprintf("%s\r\n\r\n", out.String()))); err != nil {
			return errors.Wrap(err, "file.write")
		}
	}
	file.Sync()

	return nil
}
