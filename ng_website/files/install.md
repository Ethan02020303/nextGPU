nextGPU
=======
利用闲置GPU算力为AI工作流提供服务，让你的GPU在闲暇时间也能为你赚钱。
-----------

## 准备工作：
#### 1. 有一台英伟达GPU的电脑
```json
其它类型的GPU暂时不支持
```
#### 2. 需要提前安装Ubuntu操作系统

```json
如果你用的是windows，但需要安装wsl2
```

#### 3. 在Ubuntu操作系统上安装英伟达GPU驱动
```json
在Ubuntu系统中执行指令：nvidia-smi，可以看到
```


## 手动安装运行环境

#### 在Ubuntu系统中安装docker
```json
在ubuntu系统中安装docker步骤：

​安装依赖工具:
1：sudo apt update
2：sudo apt install ca-certificates curl gnupg lsb-release

添加Docker官方GPG密钥:
1: sudo mkdir -p /etc/apt/keyrings
2: curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

设置Docker仓库:
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

安装Docker引擎:
1: sudo apt update
2: sudo apt install docker-ce docker-ce-cli containerd.io docker-compose-plugin

​验证安装:
sudo docker run hello-world
```

#### 安装docker-compose
```json
下载最新版二进制文件:
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

​授予可执行权限:
sudo chmod +x /usr/local/bin/docker-compose

验证安装:
docker-compose --version

```

## 自动安装运行环境

#### 安装方法：

- 下载nextGPU算力端安装脚本

<div align="center">
![nextGPU Screenshot](https://github.com/Ethan02020303/nextGPU/blob/main/ng_website/images/install_home.png)
</div>

[点击我](https://comfyanonymous.github.io/ComfyUI_examples)

- 运行安装脚本

```json
sudo ./install.sh
```

- 按照提示输入你的账号信息
```json


```