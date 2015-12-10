package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"gopkg.in/redis.v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"strconv"
)

func scpFile(remoteIP string, conf *ssh.ClientConfig, dest, localFile string) bool {
	client, err := ssh.Dial("tcp", remoteIP+":22", conf)
	if err != nil {
		log.Println("Scp: Fail to dial: " + err.Error())
		return false
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Println("Scp: Fail to create session: " + err.Error())
		return false
	}
	defer session.Close()

	err = scp.CopyPath(localFile, dest, session)
	if err != nil {
		log.Println("Fail to scp: " + err.Error())
		return false
	} else {
		log.Println("Success to scp")
		return true
	}
}

func md5Sum(fileName string) string {
	file, err := os.Open(fileName)
	if err != nil {
		log.Println("md5sum: Fail to open file")
		return "fail"
	}
	md5h := md5.New()
	io.Copy(md5h, file)
	return hex.EncodeToString(md5h.Sum([]byte("")))
}

func executeRemoteScript(remoteIP, passwd, scriptName string) string {
	remotePath := "/tmp/"
	scriptPath := "Script/"
	localFile := scriptPath + scriptName
	remoteFile := remotePath + scriptName

	fmt.Println("Start......")

	conf := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
	}

	var res bool
	res = scpFile(remoteIP, conf, remotePath, localFile)
	if res == false {
		return "Fail to scp script"
	}
	strMd5sum := md5Sum(localFile)
	fmt.Println(strMd5sum)

	client, err := ssh.Dial("tcp", remoteIP+":22", conf)
	defer client.Close()

	if err != nil {
		return ("Fail to dial: " + err.Error())
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a session.
	session, err := client.NewSession()
	if err != nil {
		return ("Fail to create session: " + err.Error())
	}
	// Once a session is created, you can excute a single comand on the remote side
	// using the run method.
	var b bytes.Buffer
	session.Stdout = &b

	// Comparision of md5 values
	md5Cmd := "md5sum " + remoteFile
	if err := session.Run(md5Cmd); err != nil {
		return ("Fail to run: " + err.Error())
	}
	strResult := b.String()
	if strings.HasPrefix(strResult, strMd5sum) == false {
		log.Println("md5 value different")
		return "md5 value different"
	}
	session.Close()
	b.Reset()

	// Execute remote script
	session, err = client.NewSession()
	if err != nil {
		log.Println("Fail to create session: " + err.Error())
		return "Fail to create session: " + err.Error()
	}

	session.Stdout = &b

	exeCmd := "sh " + remoteFile
	if err := session.Run(exeCmd); err != nil {
		log.Println("Fail to run: " + err.Error())
		return "Fail to run: " + err.Error()
	}

	fmt.Println(b.String())
	return b.String()
	//b.Reset()
}

func justPing(strIP string) bool {
	res, err := exec.Command("ping", "-c 1 -W 3", strIP).Output()
	if err != nil {
		fmt.Println("Exec error" + err.Error())
		return false
	}
	if strings.ContainsAny(string(res), "1 received") == true {
		return true
	} else {
		return false
	}
}

func readIPUserPasswd(fileName string) string {
	buffer, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println("Fail to read file")
	}
	return string(buffer)
}

func start(remoteIP, passwd, scriptName string, key, res chan string) {
	if justPing(remoteIP) == false {
		key <- remoteIP
		res <- "Ping error"
		return
	}
	strRes := executeRemoteScript(remoteIP, passwd, scriptName)
	key <- remoteIP
	res <- strRes
}

func main() {
	maxCPU := runtime.NumCPU() / 2
	runtime.GOMAXPROCS(maxCPU)
	scriptName := "taiyangdangkongzhao.sh"
	loginFile := "Conf/all.login"

	// Into local redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "taiyangdangkongzhao",
		DB:       0,
	})
	defer redisClient.Close()

	key := make(chan string)
	res := make(chan string)
	strFileContent := readIPUserPasswd(loginFile)
	goCount := 0
	var remoteIP string
	var userAndPasswd string

	for _, i := range strings.Split(strFileContent, "\n") {
		if strings.ContainsAny(i, "###") == true {
			s := strings.Split(i, "###")
			_, remoteIP, userAndPasswd = s[0], s[1], s[2]
			up := strings.Split(userAndPasswd, " ")
			_, passwd := up[0], up[1]
			go start(remoteIP, passwd, scriptName, key, res)
			goCount++
		}
	}

	fmt.Println(goCount)
	taskid := 0
	val, err := redisClient.Get("taskid").Result()
	if err != nil {
		fmt.Println("Fail to get taskid " + err.Error())
		redisClient.Set("taskid", taskid, 0)
	}
	taskid, _ = strconv.Atoi(val)
	taskid = taskid + 1
	redisClient.Set("taskid", taskid, 0)

	for i := 0; i < goCount; i++ {
		strKey := <-key
		strRes := <-res
		strTaskid := strconv.Itoa(taskid)
		err := redisClient.HSet(strTaskid, strKey, strRes).Err()
		if err != nil {
			fmt.Println("Redis HSet error" + err.Error())
		}
	}
}
