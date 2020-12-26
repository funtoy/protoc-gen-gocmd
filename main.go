package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
)

func errorOut(err error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.Error()
	log.Print("protoc-gen-cmd: error:", s)
	os.Exit(1)
}

func failWithMessage(msgs ...string) {
	s := strings.Join(msgs, " ")
	log.Print("protoc-gen-gocmd: error:", s)
	os.Exit(1)
}

func main() {
	g := NewGenerator()
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		errorOut(err, "reading input")
	}

	if err := proto.Unmarshal(data, g.Request); err != nil {
		errorOut(err, "parsing input proto")
	}

	if len(g.Request.FileToGenerate) == 0 {
		failWithMessage("no files to generate")
	}

	g.LoadParams()
	g.GenerateFiles()

	data, err = proto.Marshal(g.Response)
	if err != nil {
		errorOut(err, "failed to marshal output proto")
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		errorOut(err, "failed to write output proto")
	}
}
