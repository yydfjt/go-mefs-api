package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/xcshuan/go-mefs-api"
)

var sh *shell.Shell
var ncalls int

var _ = time.ANSIC

func sleep() {
	ncalls++
	//time.Sleep(time.Millisecond * 5)
}

func main() {
	sh = shell.NewShell("localhost:5001")
	//err := sh.GetObject("poss", "bucket01", path.Join(os.Getenv("HOME"), "poss1"))
	p := path.Join(os.Getenv("HOME"), "poss1")
	file, err := os.Open(p)
	ob, err := sh.PutObject(file, path.Base(file.Name()), "bucket01")
	fmt.Println(ob, err)
	bks, err := sh.ListBuckets()
	fmt.Println(bks, err)
	obs, err := sh.ListObjects(bks.Buckets[0].BucketName)
	fmt.Println(obs, err)
}
