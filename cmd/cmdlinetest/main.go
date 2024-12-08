package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"
)

// func main() {
// 	//cmd := exec.Command(".\\xelogstash.exe /config \"runall.toml\" /log /debug")
// 	var err error
// 	cmd := exec.Command("cmd", "/C", "dir")
// 	out, err := cmd.CombinedOutput()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	err = cmd.Run()
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	fmt.Println(string(out))
// 	fmt.Println("done")
// }

func main() {

	var i int
	for i = 0; i < 100000; i++ {
		log.Println("-------------------------------------------------------------------")
		log.Printf("-- Batch: %d (%s)\r\n", i, time.Now().String())
		log.Println("-------------------------------------------------------------------")
		runone()
	}
}

func runone() {

	var stdoutBuf, stderrBuf bytes.Buffer
	//cmd := exec.Command("cmd", "/C", "dir")
	cmd := exec.Command("cmd", "/C", ".\\xelogstash.exe /config \"runall.toml\" /log /debug")

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		log.Fatalf("cmd.Start() failed with '%s'\n", err)
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatal("failed to capture stdout or stderr\n")
	}
	outStr, errStr := stdoutBuf.String(), stderrBuf.String()
	fmt.Printf("\nout:\n%s\nerr:\n%s\n", outStr, errStr)
}
