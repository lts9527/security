package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"security/log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
)

type Watch struct {
	smtpPort         int
	to               []string
	from             string
	smtp             string
	smtpAuthUser     string
	smtpAuthPassword string
	Watcher          *fsnotify.Watcher
	WatchFile        WatchFile
	WatchSSH         watchSSH
}

type WatchFile struct {
	Timer      time.Duration
	watchDir   []string
	noWatchDir []string
	MsgArray   []string
	Msg        string
}

type watchSSH struct {
	CmdStrLogin        string
	CmdStrLoginWarning string
	loginIP            string
	watchSSHMsg        string
	watchSSHMsgArray   []string
}

func NewWatch() *Watch {
	// 初始化监控器
	w, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	return &Watch{
		smtpPort:         viper.GetInt("smtpPort"),
		to:               viper.GetStringSlice("to"),
		from:             viper.GetString("from"),
		smtp:             viper.GetString("smtp"),
		smtpAuthUser:     viper.GetString("smtpAuthUser"),
		smtpAuthPassword: viper.GetString("smtpAuthPassword"),
		Watcher:          w,
		WatchFile: WatchFile{
			Timer:      viper.GetDuration("WatchFile.timer"),
			watchDir:   viper.GetStringSlice("WatchFile.watchDir"),
			noWatchDir: viper.GetStringSlice("WatchFile.noWatchDir"),
		},
		WatchSSH: watchSSH{
			CmdStrLogin:        fmt.Sprintf("tail -f -n 0 %s  | grep --line-buffer '%s'", viper.GetString("watchSSH.watchDirSSH"), viper.GetString("watchSSH.loginFilterKey")),
			CmdStrLoginWarning: fmt.Sprintf("tail -f -n 0 %s | grep -B1 --line-buffer '%s'", viper.GetString("watchSSH.watchDirSSH"), viper.GetString("watchSSH.warningFilterKey")),
		},
	}
}

// BatchAdd 批量添加要监控的路径
func (w *Watch) BatchAdd() {
	for _, v := range w.WatchFile.watchDir {
		if err := w.Add(v); err != nil {
			log.Error(err.Error())
		}
	}
}

// Add添加要监控的路径
func (w *Watch) Add(watchDir string) error {
	// 遍历当前文件夹下的目录，将所有的目录添加但监听列表
	err := filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
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

// BatchDelete 批量添加不监控的路径
func (w *Watch) BatchDelete() {
	for _, v := range w.WatchFile.noWatchDir {
		if err := w.Delete(v); err != nil {
			log.Error(err.Error())
		}
	}
}

// Delete 添加不监控的路径
func (w *Watch) Delete(watchDir string) error {
	// 遍历当前文件夹下的目录，将所有的目录添加但监听列表
	err := filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
		err = w.Watcher.Remove(path)
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

// BatchSend 给多个邮箱发送信息
func (w *Watch) BatchSend() {
	for _, v := range w.WatchFile.MsgArray {
		w.WatchFile.Msg += v + "\n" + "<br>"
	}
	for _, v := range w.to {
		w.send(v, w.WatchFile.Msg, viper.GetString("WatchFile.subject"))
	}
}

func (w *Watch) send(to, msg, subject string) {
	m := gomail.NewMessage()
	//发送人
	m.SetHeader("From", w.from)
	//接收人
	m.SetHeader("To", to)
	//抄送人
	//m.SetAddressHeader("Cc", "xxx@qq.com", "xiaozhujiao")
	//主题
	m.SetHeader("Subject", subject)
	//内容
	m.SetBody("text/html", msg)
	//附件
	//m.Attach("./myIpPic.png")
	//拿到token，并进行连接,第4个参数是填授权码
	d := gomail.NewDialer(w.smtp, w.smtpPort, w.from, w.smtpAuthPassword)
	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		log.Error("DialAndSend err %v:", err)
		return
	}
	log.Info("send mail success")
}

// Screen 过滤文件，忽略包含这些关键字的文件
func (w *Watch) Screen(name string) bool {
	// 屏蔽的文件关键字
	screen := []string{".swp", ".swx", "~", "4913", "swo"}
	for _, v := range screen {
		n := strings.Contains(name, v)
		if n {
			return true
		}
	}
	for _, v := range w.WatchFile.noWatchDir {
		n := strings.Contains(name, v)
		if n {
			return true
		}
	}
	return false
}

// 执行ssh登录提醒和报警
func (w *Watch) StartWatchSSH() {
	if viper.GetBool("watchSSH.enablementSSH") {
		err := w.cmdWatchSSH(w.WatchSSH.CmdStrLogin, viper.GetString("watchSSH.loginSubject"))
		if err != nil {
			log.Error(err.Error())
		}
		err = w.cmdWatchSSH(w.WatchSSH.CmdStrLoginWarning, viper.GetString("watchSSH.warningSubject"))
		if err != nil {
			log.Error(err.Error())
		}
	}
}

func (w *Watch) cmdWatchSSH(cmd, subject string) error {
	c := exec.Command("/bin/bash", "-c", cmd)
	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	var str string
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			readString, err := reader.ReadString('\n')
			if err != nil || err == io.EOF {
				return
			}
			n := strings.Contains(readString, "maximum authentication attempts exceeded")
			log.Warn(strings.TrimSpace(readString))
			if n {
				str = "<br>" + readString
				continue
			}
			readString += "\n" + str
			for _, v := range w.to {
				w.send(v, readString, subject)
			}
		}
	}()
	err = c.Start()
	return err
}

func (w *Watch) Close() {
	err := w.Watcher.Close()
	if err != nil {
		log.Error(err.Error())
	}
}
