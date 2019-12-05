package sink

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestFileName(t *testing.T) {
	assert := assert.New(t)
	r := NewRotator("", "pfx", "log")
	r.fs = afero.NewMemMapFs()
	r.clock = clock.NewMock()
	r.ts = r.clock.Now().Format("20060102")
	assert.Equal(r.clock.Now(), time.Unix(0, 0))
	assert.Equal(r.filename(), "pfx_19691231.log")
}

func TestWrite(t *testing.T) {
	assert := assert.New(t)
	r := NewRotator("", "pfx", "log")
	fs := afero.NewMemMapFs()
	r.fs = fs
	r.clock = clock.NewMock()
	n, err := r.Write([]byte("hello world"))
	assert.NoError(err)
	assert.Equal(11, n)
	err = r.Close()
	assert.NoError(err)
	//printtree(fs)
}

func TestSubdirectory(t *testing.T) {
	assert := assert.New(t)
	r := NewRotator("events", "pfx", "log")
	fs := afero.NewMemMapFs()
	r.fs = fs
	r.clock = clock.NewMock()
	n, err := r.Write([]byte("hello world"))
	assert.NoError(err)
	assert.Equal(11, n)

	err = r.Close()
	assert.NoError(err)

	exists, err := afero.DirExists(fs, "events")
	assert.True(exists)
	assert.NoError(err)
}

func TestOpen(t *testing.T) {
	assert := assert.New(t)
	fs := afero.NewMemMapFs()
	cl := clock.NewMock()

	r := NewRotator(".", "pfx", "log")

	r.fs = fs
	r.clock = cl
	_, err := r.Write([]byte("one\r\n"))
	fname := r.file.Name()
	assert.NoError(err)
	err = r.Close()
	assert.NoError(err)

	r2 := NewRotator(".", "pfx", "log")
	r2.fs = fs
	r2.clock = cl
	_, err = r2.Write([]byte("two\r\n"))
	assert.NoError(err)
	err = r2.Close()
	assert.NoError(err)

	//printtree(fs)
	bb, err := afero.ReadFile(fs, fname)
	assert.Equal(bb, []byte("one\r\ntwo\r\n"))
}

func TestRotateDaily(t *testing.T) {
	assert := assert.New(t)
	fs := afero.NewMemMapFs()
	m := clock.NewMock()

	r := NewRotator(".", "pfx", "log")
	r.fs = fs
	r.clock = m

	_, err := r.Write([]byte("one\r\n"))
	assert.NoError(err)

	m.Add(24 * time.Hour)
	_, err = r.Write([]byte("two\r\n"))

	m.Add(24 * time.Hour)
	_, err = r.Write([]byte("three\r\n"))

	m.Add(24 * time.Hour)
	_, err = r.Write([]byte("four\r\n"))

	dd, err := afero.ReadDir(fs, ".")
	assert.NoError(err)
	assert.Equal(4, len(dd))

	assert.Equal("pfx_19691231.log", dd[0].Name())
	assert.Equal("pfx_19700101.log", dd[1].Name())
	assert.Equal("pfx_19700102.log", dd[2].Name())
	assert.Equal("pfx_19700103.log", dd[3].Name())

	//printtree(fs)
}

func TestRotateHourly(t *testing.T) {
	assert := assert.New(t)
	fs := afero.NewMemMapFs()
	m := clock.NewMock()

	r := NewRotator(".", "pfx", "log")
	r.fs = fs
	r.clock = m

	r.Hourly = true

	for i := 0; i < 100; i++ {
		_, err := r.Write([]byte("test\r\n"))
		assert.NoError(err)
		m.Add(9 * time.Minute)
	}
	//printtree(fs)
	dd, err := afero.ReadDir(fs, ".")
	assert.NoError(err)
	assert.Equal(15, len(dd))
}

func TestCleanHourly(t *testing.T) {
	dir := "zdir"
	assert := assert.New(t)
	fs := afero.NewMemMapFs()
	m := clock.NewMock()

	r := NewRotator(dir, "pfx", "log")
	r.Retention = 10 * time.Hour
	r.fs = fs
	r.clock = m
	r.Hourly = true

	for i := 0; i < 50; i++ {
		_, err := r.Write([]byte("test of whta I'm wriring!\r\n"))
		assert.NoError(err)
		m.Add(19 * time.Minute)
	}
	//printtree(fs)
	dd, err := afero.ReadDir(fs, dir)
	assert.NoError(err)
	assert.Equal(10, len(dd))
}

func TestCleanDaily(t *testing.T) {
	dir := "zdir"
	assert := assert.New(t)
	fs := afero.NewMemMapFs()
	m := clock.NewMock()

	r := NewRotator(dir, "pfx", "log")
	r.Retention = 168 * time.Hour
	r.fs = fs
	r.clock = m

	for i := 0; i < 200; i++ {
		_, err := r.Write([]byte("test of what I'm wriring!\r\n"))
		assert.NoError(err)
		m.Add(95 * time.Minute)
	}
	//printtree(fs)
	dd, err := afero.ReadDir(fs, dir)
	assert.NoError(err)
	assert.Equal(7, len(dd))
}

func printtree(fs afero.Fs) {
	println("--------------------------------------")
	println("- tree")
	println("--------------------------------------")
	err := afero.Walk(fs, ".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			fmt.Println("-", path, info.Size())
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	println("--------------------------------------")
}
