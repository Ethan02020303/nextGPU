const BASE_URL = 'https://www.nextgpu.net/node';
const TASK_URL = 'https://www.nextgpu.net/session';

/**
 * 获取并更新用户信息
 * 用于在多个页面中更新用户信息区域
 * 
 * @param {string} userName - 要查询的用户名
 */
async function sendUserInfo(userName) {   
    try {
        if (userName == "" || userName == undefined) {
            // pc
            document.getElementById('user-name').textContent = "未登录";
            document.getElementById('user-credits').textContent = 0;
            document.getElementById('user-workflows').textContent = "0/0";
            document.getElementById('user-images').textContent = 0;
            document.getElementById('user-storage').textContent = "0/0";

            const userAvatarElement = document.getElementById('user-avatar');
            userAvatarElement.style.backgroundImage = `url('images/mini_logo.png')`;
            userAvatarElement.textContent = '';

            const vipLevelElement = document.querySelector('.user-details p');
            vipLevelElement.textContent = '未知';
            vipLevelElement.style.backgroundColor = '#9e9e9e'; 

            //models
            document.getElementById('mobile-credits').textContent = 0;
            document.getElementById('mobile-workflows').textContent = "0/0";
            document.getElementById('mobile-images').textContent = 0;
            document.getElementById('mobile-storage').textContent = "0/0";
        }else{
            // 显示加载状态
            showUserLoading(true);
            const response = await fetch(`${BASE_URL}/getUser?userName=${encodeURIComponent(userName)}`, {
                method: 'GET',
            });            
            const data = await response.json();
            
            // 检查API返回的codeId
            if (data.codeId === 200 && data.userBase) {
                // 从userBase中提取用户信息
                const userData = data.userBase;
                // 1. 更新用户名
                const userNameElement = document.getElementById('user-name');
                if (userNameElement && userData.userName) {
                    userNameElement.textContent = userData.userName;
                }
                // 2. 更新用户头像
                const userAvatarElement = document.getElementById('user-avatar');
                if (userAvatarElement) {
                    if (userData.face) {
                        // 使用头像URL
                        userAvatarElement.style.backgroundImage = `url('${userData.face}')`;
                        userAvatarElement.textContent = '';
                        // 添加头像加载失败处理
                        userAvatarElement.onerror = function() {
                            this.style.backgroundImage = '';
                            const initials = userData.userName ? userData.userName.charAt(0) : '用';
                            this.textContent = initials;
                        };
                        // 确保正确显示头像
                        userAvatarElement.style.backgroundSize = 'cover';
                        userAvatarElement.style.backgroundPosition = 'center';
                    } else {
                        // 如果没有头像，使用首字母
                        const initials = userData.userName ? userData.userName.charAt(0) : '用';
                        userAvatarElement.textContent = initials;
                        userAvatarElement.style.backgroundImage = '';
                    }
                }
                // 3. 更新VIP状态（如果存在）
                const vipLevelElement = document.querySelector('.user-details p');
                if (vipLevelElement) {
                    if (userData.level === 1) {
                        vipLevelElement.textContent = 'VIP会员';
                        vipLevelElement.style.backgroundColor = '#ff9800'; // 金色
                    } else if (userData.level > 1) {
                        vipLevelElement.textContent = `SVIP${userData.level}`;
                        vipLevelElement.style.backgroundColor = '#d50000'; // 红色
                    } else {
                        vipLevelElement.textContent = '普通会员';
                        vipLevelElement.style.backgroundColor = '#9e9e9e'; // 灰色
                    }
                }
                // 4. 更新统计信息
                const credits = userData.credits || 0;
                const images = userData.images || 0;
                // 工作流数据
                const workflowText = userData.workflows && userData.workflows.used !== undefined && userData.workflows.total !== undefined
                    ? `${userData.workflows.used}/${userData.workflows.total}`
                    : '0/0';
                // 存储空间数据 (从字节转换为GB)
                const storageUsedGB = userData.storage && userData.storage.used !== undefined 
                    ? (userData.storage.used / (1024 * 1024 * 1024)).toFixed(2)
                    : 0;
                const storageTotalGB = userData.storage && userData.storage.total !== undefined 
                    ? (userData.storage.total / (1024 * 1024 * 1024)).toFixed(2)
                    : 0;
                const storageText = `${storageUsedGB}GB/${storageTotalGB}GB`;
                // 更新DOM元素
                document.getElementById('user-credits').textContent = credits;
                document.getElementById('user-workflows').textContent = workflowText;
                document.getElementById('user-images').textContent = images;
                document.getElementById('user-storage').textContent = storageText;

                document.getElementById('mobile-credits').textContent = credits;
                document.getElementById('mobile-workflows').textContent = workflowText;
                document.getElementById('mobile-images').textContent = images;
                document.getElementById('mobile-storage').textContent = storageText;

                // 隐藏加载状态
                showUserLoading(false);
            } else {
                // 非200状态码或缺少userBase数据时抛出错误
                throw new Error(`API错误: ${data.msg || '未知错误'}`);
            }

        }
    } catch (error) {
        console.error('获取用户信息失败:', error);
        // 显示错误信息
        document.querySelectorAll('.stat-value').forEach(el => {
            el.textContent = '加载失败';
        });
        // 隐藏加载状态
        showUserLoading(false);
    }
}

async function sendMyTask(userName) {
    try {
        const requestData = {
            userName: userName,
        };
        const response = await fetch(`${BASE_URL}/getMyTask?userName=${encodeURIComponent(userName)}`, {
            method: 'GET',
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data.tasks;
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        const container = document.getElementById('model-categories');
        container.innerHTML = `
        <div class="error-state">
            <i class="fas fa-exclamation-triangle"></i>
            <span>加载任务失败</span>
        </div>
        `;
        return [];
    }
}

async function sendTitle(title) {
    try {
        const requestData = {
            title: title,
        };
        const response = await fetch(`${BASE_URL}/getWorkflow`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data.workflowBase;
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        const container = document.getElementById('model-categories');
        container.innerHTML = `
        <div class="error-state">
            <i class="fas fa-exclamation-triangle"></i>
            <span>分类加载失败</span>
        </div>
        `;
        return [];
    }
}

async function sendNewTask(userName, title, imageCount, imagePath, taskID, selectedRatio, promptText) {
    try {
        const parameters = {
            imageRatio: selectedRatio,
            promptText: promptText
        };
        const requestData = {
            userName: userName,
            title: title,
            inputType: "text",
            inputImageScaledRatio: 8,
            imageCount: imageCount,
            imagePath: imagePath,
            taskID: taskID,
            data: {
                parameters: parameters
            }
        };
        const response = await fetch(`${TASK_URL}/publish`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        if (data.codeID === 200) {
            return data
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        alert(`任务提交失败: ${error.message}`);
        return [];
    }
}

async function sendRegister(userName, eMail, password) {
    try {
        const requestData = {
            userName: userName,
            eMail: eMail,
            password: password,
        };
        const response = await fetch(`${BASE_URL}/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        alert(`任务提交失败: ${error.message}`);
        return [];
    }
}

async function sendLogin(userName, password) {
    try {
        const requestData = {
            userName: userName,
            password: password,
        };
        const response = await fetch(`${BASE_URL}/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        alert(`任务提交失败: ${error.message}`);
        return [];
    }
}

// 发送订阅会员
async function sendSubscription(planId) {
    try {
        const requestData = {
            subscriptionID: planId,
        };
        const response = await fetch(`${BASE_URL}/subscription`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data.workflowBase;
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        const container = document.getElementById('model-categories');
        container.innerHTML = `
        <div class="error-state">
            <i class="fas fa-exclamation-triangle"></i>
            <span>分类加载失败</span>
        </div>
        `;
        return [];
    }
}

async function sendMyNodes(userName) {
    try {
        const requestData = {
            userName: userName,
        };
        const response = await fetch(`${BASE_URL}/getMyNode?userName=${encodeURIComponent(userName)}`, {
            method: 'GET',
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data.nodes;
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        const container = document.getElementById('model-categories');
        container.innerHTML = `
        <div class="error-state">
            <i class="fas fa-exclamation-triangle"></i>
            <span>加载任务失败</span>
        </div>
        `;
        return [];
    }
}

async function sendMyTasks(userName) {
    try {
        const requestData = {
            userName: userName,
        };
        const response = await fetch(`${BASE_URL}/getMyAllTasks?userName=${encodeURIComponent(userName)}`, {
            method: 'GET',
        });
        const data = await response.json();
        if (data.codeId === 200) {
            return data.tasks;
        } else {
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        const container = document.getElementById('model-categories');
        container.innerHTML = `
        <div class="error-state">
            <i class="fas fa-exclamation-triangle"></i>
            <span>加载任务失败</span>
        </div>
        `;
        return [];
    }
}