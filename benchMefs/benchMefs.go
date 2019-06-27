package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math/big"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ipfs/go-ipfs/data-format"
	"github.com/xcshuan/go-mefs-api"
)

var TestLog map[string]map[string]int
var sh *shell.Shell

//一些可调参数

const endPoint = "localhost:5001"

//用户数
const UserCount = 3

//每个用户上传对象数目
const ObjectCount = 1

//随机文件最大大小
const RandomDataSize = 1024 * 1024 * 100
const BucketName = "Bucket01"
const Policy = dataformat.RsPolicy
const DataCount = 3
const ParityCount = 2

//测试下载的输出路径
var outPath string

//上传下载间隔
var sleepInterval = 5 * time.Second

//用于通知结束与否
var finishChan chan struct{}

func main() {
	var err error
	rand.Seed(time.Now().Unix())
	fmt.Println("  Begin to test upload and download...")
	var UploadSuccess, Uploadfailed, DownloadSuccess, Downloadfailed int
	var UploadSize int64
	sh = shell.NewShell(endPoint)
	Users := make([]*shell.UserPrivMessage, UserCount)
	outPath = os.Getenv("GOPATH")
	finishChan = make(chan struct{}, UserCount)
	//首先创建指定数量的User
	for i := 0; i < UserCount; i++ {
		Users[i], err = sh.CreateUser()
		if err != nil {
			log.Println("Create User failed", err)
		}
		fmt.Println("  Create User", "Address", Users[i].Address, "Private Key", Users[i].Address)
		transferTo(big.NewInt(1000000000000000000), Users[i].Address)
	}

	fmt.Println("Waiting for user start...")
	for i := 0; i < UserCount; i++ {
		addr := Users[i].Address
		flag := i
		go func() {
			for {
				balance := queryBalance(addr)
				if balance.Cmp(big.NewInt(10000000000)) > 0 {
					break
				}
				fmt.Println(addr, "'s Balance now:", balance.String(), ", waiting for transfer success")
				time.Sleep(10 * time.Second)
			}
			err = sh.StartUser(addr)
			if err != nil {
				log.Println("Start User failed", err)
			}
			fmt.Println("  Begin to start User", addr)
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
				//创建一个Bucket
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
				//构造随机文件
				r := rand.Int63n(RandomDataSize)
				UploadSize += r
				data := make([]byte, r)
				fillRandom(data)
				buf := bytes.NewBuffer(data)
				objectName := addr + "_" + strconv.Itoa(int(r))
				fmt.Println("  Begin to upload", objectName, "Size is", ToStorageSize(r), "addr", addr)
				beginTime := time.Now().Unix()

				//开始上传
				ob, err := sh.PutObject(buf, objectName, BucketName, shell.SetAddress(addr))
				if err != nil {
					log.Println(addr, "Upload filed", err)
					Uploadfailed++
				}
				UploadSuccess++
				storagekb := float64(r) / 1024.0
				endTime := time.Now().Unix()
				speed := storagekb / float64(endTime-beginTime)
				fmt.Println("  Upload", objectName, "Size is", ToStorageSize(r), "speed is", speed, "KB/s", "addr", addr)
				fmt.Println(ob.String() + "address: " + addr)
				//等待一会，等上传完成
				time.Sleep(sleepInterval)
				//下面开始下载
				fmt.Println("  Begin to download", objectName, "Size is", ToStorageSize(r), "addr", addr)

				//设定输出路径
				var p string
				rootExists := true
				rootIsDir := false
				if stat, err := os.Stat(outPath); err != nil && os.IsNotExist(err) {
					rootExists = false
				} else if err != nil {
					fmt.Println("  Get object", objectName, "failed: ", "addr", addr, "err", err)
				} else if stat.IsDir() {
					rootIsDir = true
				}
				if rootIsDir == true {
					p = path.Join(outPath, objectName)
				} else if rootExists == false {
					p = outPath
				} else {
					fmt.Println("The outpath already has file: " + objectName)
				}
				var file *os.File
				if _, err := os.Stat(p); err != nil && os.IsNotExist(err) {
					file, err = os.Create(p)
					if err != nil {
						fmt.Println("  Get object", objectName, "failed: ", "addr", addr, "err", err)
					}
				} else {
					fmt.Println("The outpath already has file: " + objectName)
				}
				beginTime = time.Now().Unix()
				reader, err := sh.GetObject(objectName, BucketName, shell.SetAddress(addr))
				if err != nil {
					Downloadfailed++
					file.Close()
					fmt.Println("  Get object", objectName, "failed: ", "addr", addr, "err", err)
					continue
				}
				written, err := io.Copy(file, reader)
				if err != nil {
					Downloadfailed++
					fmt.Println("  Get object", objectName, "failed: ", "addr", addr, "err", err)
				} else {
					h := md5.New()
					newOff, err := file.Seek(0, 0)
					if err != nil {
						Downloadfailed++
						fmt.Println("  Change file Seek error", objectName, "addr", addr, "err", err, newOff)
					}
					io.Copy(h, file)
					md5Str := hex.EncodeToString(h.Sum(nil))
					if strings.Compare(md5Str, ob.Objects[0].MD5) == 0 {
						DownloadSuccess++
						fmt.Println("  Get object", objectName, "sucsess: ", ToStorageSize(r), "addr", addr)
					} else {
						Downloadfailed++
						fmt.Println("  Md5 check failed, Get", md5Str, "want", ob.Objects[0].MD5)
					}
				}
				file.Close()
				endTime = time.Now().Unix()
				storagekb = float64(written) / 1024.0
				speed = storagekb / float64(endTime-beginTime)
				fmt.Println("  Download", objectName, "Size is", ToStorageSize(r), "speed is", speed, "KB/s", "addr", addr)
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
				fmt.Println("Upload size", ToStorageSize(UploadSize))
				fmt.Printf("In this test:\nUpload %d Object success.\nUpload %d Object failed.\nDownload %d object success.\nDownload %d object failed.\n", UploadSuccess, Uploadfailed, DownloadSuccess, Downloadfailed)
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

//链的endpoint
const ethEndPoint = "http://212.64.28.207:8101"

func transferTo(value *big.Int, addr string) {
	client, err := ethclient.Dial(ethEndPoint)
	if err != nil {
		fmt.Println("rpc.Dial err", err)
		log.Fatal(err)
	}
	privateKey, err := crypto.HexToECDSA("928969b4eb7fbca964a41024412702af827cbc950dbe9268eae9f5df668c85b4")
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}
	gasLimit := uint64(21000) // in units

	gasPrice := big.NewInt(30000000000) // in wei (30 gwei)
	gasPrice, err = client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	toAddress := common.HexToAddress(addr[2:])
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("transfer ", value.String(), "to", addr)
	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
}

func queryBalance(addr string) *big.Int {
	client, err := ethclient.Dial(ethEndPoint)
	if err != nil {
		fmt.Println("rpc.Dial err", err)
		log.Fatal(err)
	}
	Address := common.HexToAddress(addr[2:])
	balance, err := client.PendingBalanceAt(context.Background(), Address)
	if err != nil {
		log.Fatal(err)
	}
	return balance
}
