package main

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"os/exec"
	"pingcheck/flaginit"
	"strings"
	"sync"
	"time"
)

// 声明全局等待组变量
var wg sync.WaitGroup

func PingCheck(sshHost, gw, sshPassword string, writer *bufio.Writer) {
	//1.ssh连接
	config := &ssh.ClientConfig{
		Timeout:         5 * time.Second, //ssh 连接time out 时间一秒钟, 如果ssh验证错误 会在一秒内返回
		User:            "root",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以, 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	config.Auth = []ssh.AuthMethod{ssh.Password(sshPassword)}
	//dial 获取ssh client
	addr := fmt.Sprintf("%s:%d", sshHost, 22)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		fmt.Printf("创建ssh client 失败:%s\n", err)
		wg.Done()
		return
	}
	defer sshClient.Close()
	//创建ssh-session
	session, err := sshClient.NewSession()
	//fmt.Println("连接成功")
	if err != nil {
		fmt.Printf("创建ssh session 失败:%s\n", err)
		wg.Done()
		return
	}
	defer session.Close()
	//2.执行远程命令
	if strings.Contains(gw, ".") {
		_, err = session.CombinedOutput(fmt.Sprintf("ping -c 4 -i 0.3 -W 5 %s", gw))
	} else if strings.Contains(gw, ":") {
		_, err = session.CombinedOutput(fmt.Sprintf("ping6 -c 4 -i 0.3 -W 5 %s", gw))
	}
	//if err != nil {
	//	log.Fatal("远程执行cmd 失败", err)
	//}
	//3.输出结果
	//log.Println("命令输出:", string(combo))
	if err != nil {
		fmt.Printf("%s ping %s down\n", sshHost, gw)
		writer.WriteString(fmt.Sprintf("%s ping %s down\n", sshHost, gw))
	} else {
		fmt.Printf("%s ping %s up\n", sshHost, gw)
		writer.WriteString(fmt.Sprintf("%s ping %s up\n", sshHost, gw))
	}
	wg.Done()
}

// 传入参数 -f 指定待ping的ip文件 -d 指定输出结果

func main() {
	//初始化flag
	checkfile, resutlfile := flaginit.InitFlag()
	//打开文件
	cf, err := os.Open(checkfile)
	if err != nil {
		log.Fatal(err)
	}
	defer cf.Close()
	rf, err := os.Create(resutlfile)
	if err != nil {
		log.Fatal(err)
	}
	defer rf.Close()
	//输入密码
	exec.Command("/bin/stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	exec.Command("/bin/stty", "-F", "/dev/tty", "-echo").Run()
	var sshPassword string
	fmt.Print("请输入密码:")
	fmt.Scan(&sshPassword)
	fmt.Println()
	exec.Command("/bin/stty", "-F", "/dev/tty", "echo").Run()
	//读入带缓冲的io
	reader := bufio.NewReader(cf)
	//写入带缓冲的io
	writer := bufio.NewWriter(rf)
	defer writer.Flush()
	for {
		host, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		//去除换行符
		host = strings.Trim(host, "\n")
		//以空格分割字符串
		arr := strings.Fields(host)
		sshHost := arr[0]
		gw := arr[1]
		wg.Add(1) // 登记1个goroutine
		go PingCheck(sshHost, gw, sshPassword, writer)
	}
	wg.Wait() // 阻塞等待登记的goroutine完成
}
