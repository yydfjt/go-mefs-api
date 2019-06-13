package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/ipfs/go-ipfs/data-format"
	"github.com/xcshuan/go-mefs-api"
)

var TestLog map[string]map[string]int
var sh *shell.Shell

//一些可调参数

//用户数
const UserCount = 3

//每个用户上传对象数目
const ObjectCount = 20

//随机文件最大大小
const RandomDataSize = 1024 * 1024 * 100
const BucketName = "Bucket01"
const Policy = dataformat.RsPolicy
const DataCount = 3
const ParityCount = 2

//测试下载的输出路径
var outPath string

//用于通知结束与否
var finishChan chan struct{}

func main() {
	fmt.Println("  Begin to test upload and download...")
	sh = shell.NewShell("localhost:5001")
	Users := make([]*shell.UserPrivMessage, UserCount)
	outPath = os.Getenv("HOME")
	finishChan = make(chan struct{}, UserCount)
	var err error
	for i := 0; i < UserCount; i++ {
		Users[i], err = sh.CreateUser()
		if err != nil {
			log.Println("Create User failed", err)
		}
		fmt.Println("  Create User", "Address", Users[i].Address, "Private Key", Users[i].Address)
		addr := Users[i].Address
		go func() {
			err = sh.StartUser(addr)
			if err != nil {
				log.Println("Start User failed", err)
			}
			fmt.Println("  Begin to start User", addr)
		}()
	}
	fmt.Println("Waiting for user start...")
	for i := 0; i < UserCount; i++ {
		addr := Users[i].Address
		flag := i
		go func() {
			//等待此User启动LFS
			for {
				err := sh.ShowStorage(shell.SetAddress(addr))
				if err != nil {
					time.Sleep(20 * time.Second)
					fmt.Println(addr, " not start, waiting..., err : ", err)
					continue
				}
				var opts []func(*shell.RequestBuilder) error
				//设置某些选项
				opts = append(opts, shell.SetAddress(addr))
				opts = append(opts, shell.SetDataCount(DataCount))
				opts = append(opts, shell.SetParityCount(ParityCount))
				if flag%2 == 0 {
					opts = append(opts, shell.SetPolicy(dataformat.RsPolicy))
				} else {
					opts = append(opts, shell.SetPolicy(dataformat.MulPolicy))
				}
				bk, err := sh.CreateBucket(BucketName, opts...)
				if err != nil {
					time.Sleep(20 * time.Second)
					fmt.Println(addr, " not start, waiting, err : ", err)
					continue
				}
				fmt.Println(bk, "addr:", addr)
				fmt.Println(addr, "started, begin to upload")
				break
			}
			//然后开始上传文件
			for j := 0; j < ObjectCount; j++ {
				r := rand.Int63n(RandomDataSize)
				data := make([]byte, r)
				fillRandom(data)
				buf := bytes.NewBuffer(data)
				objectName := addr + "_" + strconv.Itoa(int(r))
				fmt.Println("  Begin to upload", objectName, "Size is", ToStorageSize(r), "addr", addr)
				beginTime := time.Now().Unix()
				ob, err := sh.PutObject(buf, objectName, BucketName, shell.SetAddress(addr))
				if err != nil {
					log.Println(addr, "Upload filed", err)
				}
				storagekb := float64(r) / 1024.0
				endTime := time.Now().Unix()
				speed := storagekb / float64(endTime-beginTime)
				fmt.Println("  Upload", objectName, "Size is", ToStorageSize(r), "speed is", speed, "KB/s", "addr", addr)
				fmt.Println(ob.String() + "address: " + addr)

				//下面开始下载
				fmt.Println("  Begin to download", objectName, "Size is", ToStorageSize(r), "addr", addr)
				beginTime = time.Now().Unix()
				err = sh.GetObject(objectName, BucketName, outPath, shell.SetAddress(addr))
				endTime = time.Now().Unix()
				speed = storagekb / float64(endTime-beginTime)
				if err != nil {
					fmt.Println("Get object", objectName, "failed: ", "addr", addr, "err", err)
				} else {
					fmt.Println("  Download", objectName, "Size is", ToStorageSize(r), "speed is", speed, "KB/s", "addr", addr)
					fmt.Println("Get object", objectName, "sucsess: ", ToStorageSize(r), "addr", addr)
				}
			}
			fmt.Println(addr, "test finished.")
			finishChan <- struct{}{}
		}()
	}
	var finishedCount int
	for {
		select {
		case <-finishChan:
			finishedCount++
			if finishedCount >= UserCount {
				fmt.Println("all tests finished, exit...")
				return
			}
		}
	}
}
func ToStorageSize(r int64) string {
	FloatStorage := float64(r)
	var OutStorage string
	if FloatStorage < 1024 && FloatStorage >= 0 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage) + "B"
	} else if FloatStorage < 1048576 && FloatStorage >= 1024 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1024) + "KB"
	} else if FloatStorage < 1073741824 && FloatStorage >= 1048576 {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1048576) + "MB"
	} else {
		OutStorage = fmt.Sprintf("%.2f", FloatStorage/1073741824) + "GB"
	}
	return OutStorage
}

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}
