package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	c "github.com/yimiaoxiehou/minio-sync/cmd"
	"github.com/yimiaoxiehou/minio-sync/internal/minio"
)

type MyFlagSet struct {
	*flag.FlagSet
	cmdComment string // 子命令的注释
}

func main() {
	var (
		addr          string
		minioAddress  string
		minioUsername string
		minioPassword string
		skipBuckets   string
		appendOnly    string
	)

	serverCmd := &MyFlagSet{
		FlagSet:    flag.NewFlagSet("server", flag.ExitOnError),
		cmdComment: "run minio-sync server",
	}
	serverCmd.StringVar(&addr, "listen", "0.0.0.0:9010", "listen address\t\t")
	serverCmd.StringVar(&addr, "l", "0.0.0.0:9010", "listen address\t\t")
	serverCmd.StringVar(&minioAddress, "address", "127.0.0.1:9000", "minio address\t\t")
	serverCmd.StringVar(&minioAddress, "a", "127.0.0.1:9000", "minio address\t\t")
	serverCmd.StringVar(&minioUsername, "username", "minio", "minio username\t\t")
	serverCmd.StringVar(&minioUsername, "u", "minio", "minio username\t\t")
	serverCmd.StringVar(&minioPassword, "password", "minio", "minio password\t\t")
	serverCmd.StringVar(&minioPassword, "p", "minio", "minio password\t\t")

	clientCmd := &MyFlagSet{
		FlagSet:    flag.NewFlagSet("client", flag.ExitOnError),
		cmdComment: "run minio-sync client",
	}
	clientCmd.StringVar(&addr, "connect", "0.0.0.0:9010", "connect server address\t")
	clientCmd.StringVar(&addr, "c", "127.0.0.1:9010", "connect server address\t")
	clientCmd.StringVar(&minioAddress, "address", "127.0.0.1:9000", "minio address\t\t")
	clientCmd.StringVar(&minioAddress, "a", "127.0.0.1:9000", "minio address\t\t")
	clientCmd.StringVar(&minioUsername, "username", "minio", "minio username\t\t")
	clientCmd.StringVar(&minioUsername, "u", "minio", "minio username\t\t")
	clientCmd.StringVar(&minioPassword, "password", "minio", "minio password\t\t")
	clientCmd.StringVar(&minioPassword, "p", "minio", "minio password\t\t")
	clientCmd.StringVar(&skipBuckets, "skipBuckets", "false", "skip buckets\t\t")
	clientCmd.StringVar(&appendOnly, "appendonly", "false", "just sync change\t\t")

	subCmds := map[string]*MyFlagSet{"server": serverCmd, "client": clientCmd}

	useage := func() {
		fmt.Printf("Usage: minio-sync [options]\n\n")
		for _, v := range subCmds {
			fmt.Printf("%s: %s\n", v.Name(), v.cmdComment)
			v.PrintDefaults() // 使用 flag 库自带的格式输出子命令的选项帮助信息
			fmt.Println()
			fmt.Println()
			fmt.Println()
		}
		os.Exit(2)
	}

	if len(os.Args) < 2 { // 没有输入子命令, 输出帮助信息
		useage()
	}

	cmd := subCmds[os.Args[1]] //  获取子命令
	if cmd == nil {
		useage()
	}
	cmd.Parse(os.Args[2:]) // 解析子命令

	switch cmd.Name() {
	case "server":
		cmd.Visit(func(f *flag.Flag) {
			fmt.Printf("option %s, value is %s\n", f.Name, f.Value)
		})

		minio.InitMinioClient(minioAddress, minioUsername, minioPassword)
		c.RunServer(addr)

	case "client":
		cmd.Visit(func(f *flag.Flag) {
			fmt.Printf("option %s, value is %s\n", f.Name, f.Value)
		})

		minio.InitMinioClient(minioAddress, minioUsername, minioPassword)

		ao, err := strconv.ParseBool(appendOnly)
		if err != nil {
			log.Fatalln("args appendOnly parse error. mush be bool value")
		}
		c.RunClient(addr, strings.Split(skipBuckets, ","), ao)
	default:
		log.Println("Not support cmd.")
		os.Exit(2)
	}
}
