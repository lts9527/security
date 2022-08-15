package service

import (
	"os"
	"path/filepath"
	"security/log"
	"strings"

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
	MsgArray         []string
	Msg              string
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

func (w *Watch) BatchSend() {
	for _, v := range w.MsgArray {
		w.Msg += v + "\n" + "<br>"
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
	m.SetBody("text/html", w.Msg)
	//附件
	//m.Attach("./myIpPic.png")
	//拿到token，并进行连接,第4个参数是填授权码
	d := gomail.NewDialer(w.smtp, w.smtpPort, w.from, w.smtpAuthPassword)
	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		log.Error("DialAndSend err %v:", err)
		return
	}
	log.Info("send mail success\n")
}

func (w *Watch) Screen(name string) bool {
	// 屏蔽的文件关键字
	screen := []string{".swp", ".swx", "~", "4913"}
	for _, v := range screen {
		n := strings.Contains(name, v)
		if n {
			return true
		}
	}
	for _, v := range w.noWatchDir {
		n := strings.Contains(name, v)
		if n {
			return true
		}
	}
	return false
}

func (w *Watch) Close() {
	w.Watcher.Close()
}
