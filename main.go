package main

import (
	"fmt"
	"os"
	"path/filepath"
	"test/config"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"gopkg.in/gomail.v2"
)

type Watch struct {
	timer            int
	smtpPort         int
	to               []string
	watchdir         []string
	noWatchDir       []string
	msgArray         []string
	msg              string
	from             string
	smtp             string
	smtpAuthUser     string
	smtpAuthPassword string
	Watcher          *fsnotify.Watcher
}

func NewWarch() *Watch {
	// 初始化监控器
	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	return &Watch{
		timer:            viper.GetInt("timer"),
		smtpPort:         viper.GetInt("smtpPort"),
		to:               viper.GetStringSlice("to"),
		from:             viper.GetString("from"),
		watchdir:         viper.GetStringSlice("watchDir"),
		noWatchDir:       viper.GetStringSlice("noWatchDir"),
		smtp:             viper.GetString("smtp"),
		smtpAuthUser:     viper.GetString("smtpAuthUser"),
		smtpAuthPassword: viper.GetString("smtpAuthPassword"),
		Watcher:          w,
	}
}

func (w *Watch) BatchAdd() {
	for _, v := range w.watchdir {
		w.Add(v)
	}
}

func (w *Watch) Add(watchdir string) error {
	// 遍历当前文件夹下的目录，将所有的目录添加但监听列表
	err := filepath.Walk(watchdir, func(path string, info os.FileInfo, err error) error {
		err = w.Watcher.Add(path)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (w *Watch) BatchDelete() {
	for _, v := range w.noWatchDir {
		w.Delete(v)
	}
}

func (w *Watch) Delete(watchdir string) error {
	err := w.Watcher.Remove(watchdir)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watch) batchSend() {
	for _, v := range w.msgArray {
		w.msg += v + "\n" + "<br>"
	}
	for _, v := range w.to {
		w.send(v)
	}
}

func (w *Watch) send(to string) {
	m := gomail.NewMessage()
	//发送人
	m.SetHeader("From", w.from)
	//接收人
	m.SetHeader("To", to)
	//抄送人
	//m.SetAddressHeader("Cc", "xxx@qq.com", "xiaozhujiao")
	//主题
	m.SetHeader("Subject", viper.GetString("subject"))
	//内容
	m.SetBody("text/html", w.msg)
	//附件
	//m.Attach("./myIpPic.png")
	//拿到token，并进行连接,第4个参数是填授权码
	d := gomail.NewDialer(w.smtp, w.smtpPort, w.from, w.smtpAuthPassword)
	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		fmt.Printf("DialAndSend err %v:", err)
		return
	}
	fmt.Printf("send mail success\n")
}

func (w *Watch) Close() {
	w.Watcher.Close()
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("panic: %#v\n", err)
		}
	}()
	config.InitConfig()
	// 定时器
	ticker := time.NewTicker(time.Minute * 1)
	// 初始化监控器
	w := NewWarch()
	w.BatchAdd()
	w.BatchDelete()
	for {
		select {
		case ev := <-w.Watcher.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					// 当为文件创建时
					// fmt.Println(ev.Name, "created!!!")
					// 判断是否是文件夹
					info, err := os.Stat(ev.Name)
					if err != nil {
						fmt.Println(err)
					}
					// 如果是文件夹，添加到侦听列表
					if info.IsDir() {
						w.Add(ev.Name)
					}
				}
				if ev.Op&fsnotify.Create == fsnotify.Create {
					// 修改权限
					fmt.Println(ev.Name, "create")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: create")
					w.msgArray = append(w.msgArray, str)
					continue
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					// 文件修改
					// fmt.Println(ev.Name, "changed")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: changed")
					w.msgArray = append(w.msgArray, str)
					continue
				}
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					// 修改权限
					// fmt.Println(ev.Name, "chmod")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: chmod")
					w.msgArray = append(w.msgArray, str)
					continue
				}
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					// 文件删除
					fmt.Println(ev.Name, "removed")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: removed")
					w.msgArray = append(w.msgArray, str)
					continue
				}
			}
		case <-ticker.C:
			if len(w.msgArray) == 0 && len(w.msg) == 0 {
				fmt.Println("消息内容为空")
				continue
			}
			fmt.Println("消息内容 ", w.msg)
			w.batchSend()
			w.msgArray = w.msgArray[0:0]
			w.msg = ""
		case err := <-w.Watcher.Errors:
			{
				fmt.Println(err)
				return
			}
		}
	}
}
