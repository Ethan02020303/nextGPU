// 添加导航函数
function navigateToDetailPage(model) {
    console.log("click navigateToDetailPage");
    sessionStorage.setItem('selectedWorkflow', JSON.stringify(model));

    const encodeParam = encodeURIComponent(model.workflowTitle)
    window.location.href = `generate.html?title=${encodeParam}`;
}

function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// 渲染卡片到页面
async function renderNodes(nodes) {
    const container = document.getElementById('projects-container');
    const loader = document.getElementById('loader');
    const errorEl = document.getElementById('error');
    const template = document.getElementById('github-card-template');
    
    // 隐藏错误消息，显示加载状态
    errorEl.style.display = 'none';
    loader.style.display = 'block';
    container.innerHTML = '';
    
    try {
        // 为每个卡片数据项创建卡片
        nodes.forEach((data, index) => {
            // 克隆模板内容
            const card = template.content.cloneNode(true);
            
            // 设置状态
            const status = card.querySelector('.node-status');
            status.textContent = data.state >= 3 ? '在线' : '离线';
            status.className = 'node-status ' + (data.state >= 3 ? 'online' : 
                                              data.ststateatus === '忙碌' ? 'busy' : 'offline');
            
            card.querySelector('.ip-highlight').textContent = data.nodeIP;
            card.querySelector('.systemUUID').textContent = data.systemUUID;
            
            // 设置注册时间和最后上线时间
            card.querySelector('.register-time .value').textContent = data.registerTime;
            card.querySelector('.last-online .value').textContent = data.enrollTime;
            
            // 设置统计数据                      
            card.querySelector('.runtime .value').innerHTML = 
                `<span class="highlight-number">${formatNumber(data.onlineTime)}</span> <span class="unit">秒</span>`;
            card.querySelector('.tasks .value').innerHTML = 
                `<span class="highlight-number task-count">${data.tasks}</span>`;
            card.querySelector('.gpu-time .value').innerHTML = 
                `<span class="highlight-number">${formatNumber(data.gpuTime)}</span> <span class="unit">毫秒</span>`;
            card.querySelector('.cur-queue .value').innerHTML = 
                `<span class="highlight-number">${data.curTasks}</span>`;

            if(data.os) {
                const cleanOS = data.os.split('(')[0].trim();
                card.querySelector('.os-version').innerHTML = 
                    `<span class="blue-highlight">${cleanOS}</span>`;
            } else {
                card.querySelector('.os-version').textContent = '未知系统';
            }
            if(data.gpu) {
                const gpuParts = data.gpu;
                card.querySelector('.gpu-details').innerHTML = 
                    `<span class="blue-highlight">${gpuParts}</span>`;
            } else {
                card.querySelector('.gpu-details').textContent = '未知类型';
            }

            // 设置状态徽章
            const recommended = card.querySelector('.recommended');
            recommended.style.display = 'none';
            
            // 设置卡片出现动画
            const cardEl = card.querySelector('.github-card');
            setTimeout(() => {
                cardEl.style.opacity = '1';
                cardEl.style.transform = 'translateY(0)';
            }, index * 200);
            
            // 添加到容器
            container.appendChild(card);
        });
        
        // 添加动作按钮事件监听器
        setTimeout(() => {
            const actionButtons = document.querySelectorAll('.action-btn');
            actionButtons.forEach(btn => {
                btn.addEventListener('click', function(e) {
                    e.stopPropagation();
                    const originalHTML = this.innerHTML;
                    this.innerHTML = '✓';
                    this.style.backgroundColor = 'rgba(101, 163, 13, 0.3)';
                    this.style.borderColor = 'rgba(101, 163, 13, 0.7)';
                    
                    setTimeout(() => {
                        this.innerHTML = originalHTML;
                        this.style.backgroundColor = '';
                        this.style.borderColor = '';
                    }, 1200);
                });
            });
        }, 100);
        
    } catch (error) {
        errorEl.style.display = 'block';
        container.innerHTML = '';
        console.error("渲染卡片出错:", error);
    } finally {
        loader.style.display = 'none';
    }
}

// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', async function() {
    try {
        //获取用户信息
        const {logined, userName} = checkLoginStatus();
        if(logined){
            sendUserInfo(userName)
        }else{
            sendUserInfo()
        }
        // 渲染算力卡片
        const nodes = await sendMyNodes(userName);
        renderNodes(nodes);
    } catch (error) {
        console.error('初始化失败:', error);
        document.getElementById('models-container').innerHTML = `
            <div class="error-state">
                <i class="fas fa-exclamation-triangle fa-3x"></i>
                <p>数据加载失败，请稍后再试</p>
            </div>
        `;
    }
});