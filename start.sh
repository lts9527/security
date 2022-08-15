#!/bin/bash
#fonts color
Green="\033[32m"
Red="\033[31m"
GreenBG="\033[42;37m"
RedBG="\033[41;37m"
Font="\033[0m"

#notification information
OK="${Green}[OK]${Font}"
Error="${Red}[错误]${Font}"

cmdPath=`pwd`
mkdir -p $cmdPath
passwd1=`date +%s | sha256sum | base64 | head -c 32`

source '/etc/os-release' > /dev/null

if [ -f "/usr/bin/yum" ] && [ -d "/etc/yum.repos.d" ]; then
    PM="yum"
elif [ -f "/usr/bin/apt-get" ] && [ -f "/usr/bin/dpkg" ]; then
    PM="apt-get"        
fi

judge() {
    if [[ 0 -eq $? ]]; then
        echo -e "${OK} ${GreenBG} $1 完成 ${Font}"
        sleep 1
    else
        echo -e "${Error} ${RedBG} $1 失败 ${Font}"
        exit 1
    fi
}

is_root() {
    if [ 0 == $UID ]; then
        echo -e "${OK} ${GreenBG} 当前用户是root用户，权限正常... ${Font}"
        sleep 1
    else
        echo -e "${Error} ${RedBG} 当前用户不是root用户，请切换到root用户后重新执行脚本 ${Font}"
        exit 0.5
    fi
}

check_system() {
    if [[ "${ID}" = "centos" && ${VERSION_ID} -ge 7 ]]; then
        echo > /dev/null
    elif [[ "${ID}" = "debian" && ${VERSION_ID} -ge 8 ]]; then
        echo > /dev/null
    elif [[ "${ID}" = "ubuntu" && $(echo "${VERSION_ID}" | cut -d '.' -f1) -ge 16 ]]; then
        echo > /dev/null
    else
        echo -e "${Error} ${RedBG} 当前系统为 ${ID} ${VERSION_ID} 不在支持的系统列表内，安装中断 ${Font}"
        rm -f $cmdPath
        exit 1
    fi
    #
    if [ "${PM}" = "yum" ]; then
        sudo yum install -y epel-release
    fi
}

check_docker() {
    docker --version &> /dev/null
    if [ $? -ne  0 ]; then
        echo -e "安装docker环境..."
        curl -sSL https://get.daocloud.io/docker | sh
        echo -e "${OK} Docker环境安装完成！"
    fi
    systemctl start docker
    if [[ 0 -ne $? ]]; then
        echo -e "${Error} ${RedBG} Docker 启动 失败${Font}"
        rm -f $cmdPath
        exit 1
    fi
    #
    docker-compose --version &> /dev/null
    if [ $? -ne  0 ]; then
        echo -e "安装docker-compose..."
        curl -s -L "https://get.daocloud.io/docker/compose/releases/download/v2.5.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
        ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
        echo -e "${OK} Docker-compose安装完成！"
        service docker restart
    fi
}

 mailInstall(){
    if [ "${PM}" = "yum" ]; then
        sudo yum update -y
        sudo yum install -y curl wget nss-tools openssl openssl-devel inotify-tools
        sudo yum install -y mailx
    elif [ "${PM}" = "apt-get" ]; then
        apt-get update -y
        apt-get install -y curl wget  libnss3-tools openssl libssl-dev inotify-tools
        apt-get install -y mailutils
    fi
    judge "安装脚本依赖"
    #
    service sendmail stop > /dev/null 2>&1
    chkconfig sendmail off > /dev/null 2>&1
    service postfix start > /dev/null 2>&1
    chkconfig postfix on > /dev/null 2>&1
    postfix check > /dev/null 2>&1
    systemctl status postfix 
    echo "inet_interfaces = all" >> /etc/postfix/main.cf
    mkdir -p /root/.certs/
    cd .certs/
    echo -n | openssl s_client -connect smtp.qq.com:465 | sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > ~/.certs/qq.crt
    certutil -A -n "GeoTrust SSL CA" -t "C,," -d ~/.certs -i ~/.certs/qq.crt
    certutil -A -n "GeoTrust Global CA" -t "C,," -d ~/.certs -i ~/.certs/qq.crt
    certutil -L -d /root/.certs
    certutil -A -n "GeoTrust SSL CA - G3" -t "Pu,Pu,Pu" -d ./ -i qq.crt
    echo "set from=2622788078@qq.com
    set smtp=smtps://smtp.qq.com:465
    set smtp-auth-user=2622788078@qq.com
    set smtp-auth-password=qzxdovqtzvrwecbe
    set smtp-auth=login
    set ssl-verify=ignore" >> /etc/mail.rc
    echo "fs.inotify.max_queued_events = 16384
    fs.inotify.max_user_instances = 1024
    fs.inotify.max_user_watches = 1048576" >> /etc/sysctl.conf
}

ssh_port_exist_check() {
    if [[ 0 -eq $(lsof -i:"$1"  | grep -i -c "listen") ]]; then
        echo -e "${OK} ${GreenBG} $1 端口未被占用 ${Font}"
        sleep 1
    else
        echo -e "${Error} ${RedBG} 检测到 $1 端口被占用，以下为 $1 端口占用信息 ${Font}"
        lsof -i:"$1"
        # echo -e "${OK} ${GreenBG} 5s 后将尝试自动更换占用端口为1967 ${Font}"
        # sleep 5
        # sed -i 's/#Port 22/Port 1967/' /etc/ssh/sshd_config
        # systemctl restart sshd
        # echo -e "${OK} ${GreenBG} 更换完成 ${Font}"
        sleep 3
        echo -e "${Error} ${RedBG} 请自行更换端口后再执行程序再次安装 ${Font}"
        sleep 1
        exit
    fi
}

inotifyWait() {
    chmod +x $cmdPath/main
    kill -9 $(pidof $cmdPath/main)
    $cmdPath/main &>> $cmdPath/log/fileChange.log &
}

 sshHoneypot() {
    ssh_port_exist_check 22
    if [0 -eq docker ps |grep hfish |wc -l];then
        echo -e "${Error} ${RedBG} 检测到目前蜜罐程序已安装，是否卸载重装？（Y/n）${Font}"
        read -r uninstall_install
        [[ -z ${uninstall_install} ]] && uninstall_install="Y"
        case $uninstall_install in
        [yY][eE][sS] | [yY])
            echo -e "${GreenBG} 继续安装 ${Font}"
            sleep 1
            ;;
        *)
            echo -e "${RedBG} 安装终止 ${Font}"
            exit 2
            ;;
        esac
    fi
    docker run -d --name hfish  -p 22:22 -p 9001:9001 --restart=always imdevops/hfish:latest
    docker exec -it hfish sh -c "cp /opt/HFish/config.ini /opt/HFish/config.ini.bak"   
    docker exec -it hfish sh -c "sed -i '9s/.*/password = $passwd1/' /opt/HFish/config.ini"
    docker restart hfish
    if [0 -eq docker ps |grep hfish |wc -l];then
        echo -e "ssh蜜罐程序部署成功，请访问ip:9001进行管理"
        echo -e "账号：admin"
        echo -e "密码：$passwd1"
    fi
 }
 iptablesIN() {
    while [ -z "$baninip" ]; do
        read -rp "请输入您想禁止的ip(例如:192.168.1.1/24):" baninip
    done
    iptables -I INPUT -s $baninip -j DROP
    echo "iptables -D INPUT -s $baninip -j DROP" >> $cmdPath/.Networkban.ip
 }

 iptablesOUT() {
    while [ -z "$banoutip" ]; do
        read -rp "请输入您想禁止的ip(例如:192.168.1.1/24):" banoutip
    done
    iptables -I OUTPUT -s $banoutip -j DROP
    echo "iptables -D OUTPUT -s $banoutip -j DROP" >> $cmdPath/.Networkban.ip
 }

 iptablesdd(){
    echo -e "${OK} ${GreenBG} 1.禁止该ip数据离开本机 ${Font}"
    echo -e "${OK} ${GreenBG} 2.禁止该ip数据进入本机 ${Font}"
    read -r iptabless
    [[ -z ${iptabless} ]] && iptabless="2"
    if [ $iptabless -eq 1 ];then
        iptablesOUT
    else
        iptablesIN
    fi
 }

        if [[ "${ID}" == "ubuntu" ]] ||  [[ "${ID}" == "debian" ]];then
            echo "*/1 * * * * /bin/bash $cmdPath/.inotifyWaitDetection.sh" >>/var/spool/cron/crontabs/root
        elif [[ "${ID}" == "centos" ]];then
            echo "*/1 * * * * /bin/bash $cmdPath/.inotifyWaitDetection.sh" >>/var/spool/cron/root
        else
            echo -e "${Error} ${RedBG} 当前系统为 ${ID} ${VERSION_ID} 不在支持的系统列表内，安装中断 ${Font}"
            exit 1
        fi

 sshLoginLog() {
    if [[ "${ID}" = "centos" && ${VERSION_ID} -ge 7 ]]; then
        tail -f /var/log/secure | grep 'Accepted'| grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshLogin.ip &
    elif [[ "${ID}" = "debian" && ${VERSION_ID} -ge 8 ]]; then
        tail -f /var/log/auth.log | grep 'Accepted'| grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshLogin.ip &
    elif [[ "${ID}" = "ubuntu" && $(echo "${VERSION_ID}" | cut -d '.' -f1) -ge 16 ]]; then
        tail -f /var/log/auth.log | grep 'Accepted'| grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshLogin.ip &
    else
        echo -e "${Error} ${RedBG} 当前系统为 ${ID} ${VERSION_ID} 不在支持的系统列表内，安装中断 ${Font}"
        exit 1
    fi
 }

 sshBoomLog() {
    if [[ "${ID}" = "centos" && ${VERSION_ID} -ge 7 ]]; then
        tail -f /var/log/secure |  grep -B1 'Too many authentication failures' | grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshBoom.ip &
    elif [[ "${ID}" = "debian" && ${VERSION_ID} -ge 8 ]]; then
        tail -f /var/log/auth.log |  grep -B1 'Too many authentication failures' | grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshBoom.ip &
    elif [[ "${ID}" = "ubuntu" && $(echo "${VERSION_ID}" | cut -d '.' -f1) -ge 16 ]]; then
        tail -f /var/log/auth.log |  grep -B1 'Too many authentication failures' | grep -Po '(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|[1-9])(\.(1\d{2}|2[0-4]\d|25[0-5]|[1-9]\d|\d)){3}' &> $cmdPath/log/sshBoom.ip &
    else
        echo -e "${Error} ${RedBG} 当前系统为 ${ID} ${VERSION_ID} 不在支持的系统列表内，安装中断 ${Font}"
        exit 1
    fi
 }

show_menu() {
    echo -e "—————————— 安装向导 ——————————"
    echo -e "${Green}A.${Font}  首次一键安装环境"
    echo -e "${Green}B.${Font}  开启监控目录功能"
    echo -e "${Green}C.${Font}  开启ssh蜜罐功能"
    echo -e "${Green}D.${Font}  开启ssh登录提醒与爆破预警功能"
    echo -e "${Green}E.${Font}  开启ip封禁功能"
    echo -e "${Green}Z.${Font}  退出脚本 \n"

    read -rp "请输入代码：" menu_num
    for menu_index in `seq 0 $((${#menu_num}-1))`
    do
        case $(echo "${menu_num:$menu_index:1}" | tr "a-z" "A-Z") in
        Z)
            exit 0
            ;;
        A)
            is_root
            check_system
            check_docker
            mailInstall
            ;;
        B)
            is_root
            inotifyWait
            ;;
        C)
            is_root
            sshHoneypot
            ;;
        D)
            is_root
            sshLoginLog
            sshBoomLog
            ;;  
        E)
            is_root
            iptablesdd
            ;;
        *)
            echo -e "${RedBG}请输入正确的操作代码${Font}"
            ;;
        esac
    done
}

show_menu

