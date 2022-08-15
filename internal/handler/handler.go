package handler

import (
	"fmt"
	"os"
	"security/internal/service"
	"security/log"
	"time"

	"github.com/spf13/viper"

	"github.com/fsnotify/fsnotify"
)

func Handier() {
	// 定时器
	ticker := time.NewTicker(time.Minute * viper.GetDuration("timer"))
	// 初始化监控器
	w := service.NewWarch()
	defer w.Close()
	w.BatchAdd()
	w.BatchDelete()
	for {
		select {
		case ev := <-w.Watcher.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					if w.Screen(ev.Name) {
						continue
					}
					// 当为文件创建时
					log.Warn(ev.Name+" %s ", "created")
					// 判断是否是文件夹
					info, err := os.Stat(ev.Name)
					if err != nil {
						log.Error(err.Error())
					}
					// 如果是文件夹，添加到侦听列表
					if info.IsDir() {
						w.Add(ev.Name)
					}
				}
				if ev.Op&fsnotify.Create == fsnotify.Create {
					if w.Screen(ev.Name) {
						continue
					}
					// 修改权限
					log.Warn(ev.Name+" %s ", "create")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: create")
					w.MsgArray = append(w.MsgArray, str)
					continue
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					if w.Screen(ev.Name) {
						continue
					}
					// 文件修改
					log.Warn(ev.Name+" %s ", "changed")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: changed")
					w.MsgArray = append(w.MsgArray, str)
					continue
				}
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					if w.Screen(ev.Name) {
						continue
					}
					// 修改权限
					log.Warn(ev.Name+" %s ", "chmod")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: chmod")
					w.MsgArray = append(w.MsgArray, str)
					continue
				}
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					if w.Screen(ev.Name) {
						continue
					}
					// 文件删除
					log.Warn(ev.Name+" %s ", "removed")
					str := fmt.Sprintf("%s", "文件路径: "+ev.Name+" 操作类型: removed")
					w.MsgArray = append(w.MsgArray, str)
					continue
				}
			}
		case <-ticker.C:
			if len(w.MsgArray) == 0 && len(w.Msg) == 0 {
				continue
			}
			w.BatchSend()
			w.MsgArray = w.MsgArray[0:0]
			w.Msg = ""
		case err := <-w.Watcher.Errors:
			{
				log.Error(err.Error())
				return
			}
		}
	}
}
