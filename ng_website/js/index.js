const DOM_ELEMENTS = {
    dailyTasks: document.getElementById('daily-tasks'),
    availableNodesCount: document.getElementById('available-gpu-nodes-count'),
    nodesTableBody: document.getElementById('gpu-nodes-table-body')
};

async function fetchTaskCount() {
    try {
        const response = await fetch(`${BASE_URL}/getTaskCount`, {
            method: 'GET',
        });
        const data = await response.json();
        // 检查API返回的codeId
        if (data.codeId === 200) {
            // 返回categories数组
            DOM_ELEMENTS.dailyTasks.textContent = data.taskCount.toLocaleString();
        } else {
            // 非200状态码时抛出错误
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }        
    } catch (error) {
        console.error('获取每日任务数量失败:', error);
        DOM_ELEMENTS.dailyTasks.textContent = '加载失败';
    }
}

async function fetchNodeCount() {
    try {
        const response = await fetch(`${BASE_URL}/getNodeCount`, {
            method: 'GET',
        });
        const data = await response.json();
        // 检查API返回的codeId
        if (data.codeId === 200) {
            // 返回categories数组
            DOM_ELEMENTS.availableNodesCount.textContent = data.nodeCount.toLocaleString();
        } else {
            // 非200状态码时抛出错误
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }        
    } catch (error) {
        console.error('获取每日任务数量失败:', error);
        DOM_ELEMENTS.dailyTasks.textContent = '加载失败';
    }
}

// 重试加载数据
function retryLoading() {
    DOM_ELEMENTS.nodesTableBody.innerHTML = `
        <tr class="loading-row">
            <td colspan="5">
                <div class="loading">
                    <div class="spinner"></div>
                    <span>正在重新加载数据...</span>
                </div>
            </td>
        </tr>
    `;
    loadData();
}

// 获取可用GPU节点
async function fetchCurNodes() {
    try {
        const response = await fetch(`${BASE_URL}/getCurNode`, {
            method: 'GET',
        });
        const data = await response.json();
        // DOM_ELEMENTS.availableNodesCount.textContent = data.nodes.length;
        DOM_ELEMENTS.nodesTableBody.innerHTML = '';
        if (data.nodes.length === 0) {
            DOM_ELEMENTS.nodesTableBody.innerHTML = `
                <tr>
                    <td colspan="5" class="empty-row">
                        <div class="loading">
                            <i class="fas fa-exclamation-circle" style="color: var(--warning); font-size: 1.5rem;"></i>
                            <span>当前没有可用节点</span>
                        </div>
                    </td>
                </tr>
            `;
            return;
        }
        
        // 填充节点数据
        data.nodes.forEach(node => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${node.gpu}</td>
                <td>${node.memory} GB</td>
                <td>${node.gpuDriver}</td>
                <td>${node.onlineTime} 秒</td>
                <td>${node.tasksCount}</td>
            `;
            DOM_ELEMENTS.nodesTableBody.appendChild(row);
        });
        
    } catch (error) {
        console.error('获取GPU节点失败:', error);
        
        DOM_ELEMENTS.nodesTableBody.innerHTML = `
            <tr class="error-row">
                <td colspan="5">
                    <div class="error">
                        <i class="fas fa-exclamation-triangle"></i>
                        <p>加载节点数据失败: ${error.message}</p>
                        <button class="retry-btn" onclick="retryLoading()">
                            <i class="fas fa-redo"></i> 重试
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }
}


// 加载所有数据
function loadData() {
    fetchTaskCount();
    fetchNodeCount();
    fetchCurNodes();
}


document.addEventListener('DOMContentLoaded', async function() {
    //加载数据
    loadData();

    //判断用户是否已经登录
    const {logined, userName} = checkLoginStatus();
    if (logined) {
        loginMessage.style.display = 'none';
        document.querySelector('.auth-buttons').innerHTML = `
            <a href="#" class="btn btn-primary">
                <i class="fas fa-user-circle"></i> ${username}
            </a>
        `;
    }

    // 移动端菜单切换
    const mobileMenuBtn = document.querySelector('.mobile-menu-btn');
    const nav = document.querySelector('nav');

    mobileMenuBtn.addEventListener('click', function() {
        nav.classList.toggle('active');
        const icon = mobileMenuBtn.querySelector('i');
        if (nav.classList.contains('active')) {
            icon.classList.remove('fa-bars');
            icon.classList.add('fa-times');
        } else {
            icon.classList.remove('fa-times');
            icon.classList.add('fa-bars');
        }
    });

    // FAQ手风琴效果
    const faqItems = document.querySelectorAll('.faq-item');
    faqItems.forEach(item => {
        const question = item.querySelector('.faq-question');
        question.addEventListener('click', () => {
            const isActive = item.classList.contains('active');
            // 先关闭所有项目
            faqItems.forEach(faq => {
                faq.classList.remove('active');
                faq.querySelector('.faq-question i').classList.remove('fa-chevron-up');
                faq.querySelector('.faq-question i').classList.add('fa-chevron-down');
            });
            // 打开点击的项目
            if (!isActive) {
                item.classList.add('active');
                question.querySelector('i').classList.remove('fa-chevron-down');
                question.querySelector('i').classList.add('fa-chevron-up');
            }
        });
    });

    // 返回顶部功能
    const scrollTopBtn = document.querySelector('.scroll-top');
    window.addEventListener('scroll', () => {
        if (window.scrollY > 500) {
            scrollTopBtn.classList.add('active');
        } else {
            scrollTopBtn.classList.remove('active');
        }
    });
    scrollTopBtn.addEventListener('click', () => {
        window.scrollTo({
            top: 0,
            behavior: 'smooth'
        });
    });

    // 滚动时头部阴影效果
    window.addEventListener('scroll', () => {
        const header = document.querySelector('header');
        if (window.scrollY > 10) {
            header.style.boxShadow = '0 4px 12px rgba(0, 0, 0, 0.1)';
        } else {
            header.style.boxShadow = '0 2px 10px rgba(0, 0, 0, 0.1)';
        }
    });

    // 锚链接平滑滚动
    // document.querySelectorAll('a[href^="#"]').forEach(anchor => {
    //     anchor.addEventListener('click', function(e) {
    //         e.preventDefault();
    //         const target = document.querySelector(this.getAttribute('href'));
    //         if (target) {
    //             window.scrollTo({
    //                 top: target.offsetTop - 70,
    //                 behavior: 'smooth'
    //             });
    //         }
    //     });
    // });




    // 获取DOM元素
    const loginModal = document.getElementById('loginModal');
    const registerModal = document.getElementById('registerModal');
    const loginBtn = document.getElementById('loginBtn');
    const registerBtn = document.getElementById('registerBtn');
    const closeButtons = document.querySelectorAll('.close-modal');
    const switchToRegister = document.getElementById('switchToRegister');
    const switchToLogin = document.getElementById('switchToLogin');
    const loginForm = document.getElementById('loginForm');
    const registerForm = document.getElementById('registerForm');
    const loginMessage = document.getElementById('loginMessage');
    const registerMessage = document.getElementById('registerMessage');

    // 显示登录模态框
    function showLoginModal() {
        registerModal.style.display = 'none';
        loginModal.style.display = 'block';
        document.body.style.overflow = 'hidden';
    }

    // 显示注册模态框
    function showRegisterModal() {
        loginModal.style.display = 'none';
        registerModal.style.display = 'block';
        document.body.style.overflow = 'hidden';
    }

    // 关闭模态框
    function closeModal() {
        loginModal.style.display = 'none';
        registerModal.style.display = 'none';
        document.body.style.overflow = 'auto';
    }

    // 事件监听器
    loginBtn.addEventListener('click', (e) => {
        e.preventDefault();
        showLoginModal();
    });

    registerBtn.addEventListener('click', (e) => {
        e.preventDefault();
        showRegisterModal();
    });

    closeButtons.forEach(button => {
        button.addEventListener('click', closeModal);
    });

    window.addEventListener('click', (e) => {
        if (e.target === loginModal || e.target === registerModal) {
            closeModal();
        }
    });

    switchToRegister.addEventListener('click', (e) => {
        e.preventDefault();
        showRegisterModal();
    });

    switchToLogin.addEventListener('click', (e) => {
        e.preventDefault();
        showLoginModal();
    });

    // 登录表单提交
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;
        
        // 模拟登录验证
        if (username && password) {
            const res = await sendLogin(username, password);
            if(res.codeId === 200) {
                setLoginStatus(true, username)
                loginMessage.style.display = 'none';
                document.querySelector('.auth-buttons').innerHTML = `
                    <a href="#" class="btn btn-primary">
                        <i class="fas fa-user-circle"></i> ${username}
                    </a>
                `;
                setTimeout(() => {
                    closeModal();
                    // alert(`欢迎回来，${username}！您已成功登录。`);
                }, 500);
            }else{
                alert('账号或者密码不正确，请重新登录');
                loginForm.reset();
            }            
        } else {
            // 显示错误信息
            loginMessage.style.display = 'block';
        }
    });

    // 注册表单提交
    registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const fullname = document.getElementById('fullname').value;
        const email = document.getElementById('email').value;
        const password = document.getElementById('newPassword').value;
        const confirmPassword = document.getElementById('confirmPassword').value;
        if (password !== confirmPassword) {
            alert('两次输入的密码不一致！');
            return;
        }
        const res = await sendRegister(fullname, email, password);
        if(res.success) {
            registerMessage.style.display = 'block';
            registerForm.reset();
            setTimeout(() => {
                showLoginModal();
                registerMessage.style.display = 'none';
            }, 3000);
        }else{
            registerForm.reset();
        }
    });

    // 社交登录按钮事件
    document.querySelectorAll('.social-btn').forEach(button => {
        button.addEventListener('click', () => {
            const provider = button.classList.contains('google') ? 'Google' : 
                            button.classList.contains('github') ? 'GitHub' : 'Microsoft';
            alert(`您选择了通过${provider}账号登录`);
        });
    });
});