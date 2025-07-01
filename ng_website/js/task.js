function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

async function renderTasks(tasks) {
    const container = document.getElementById('projects-container');
    const loader = document.getElementById('loader');
    const errorEl = document.getElementById('error');
    const template = document.getElementById('task-card-template');
    
    errorEl.style.display = 'none';
    loader.style.display = 'block';
    container.innerHTML = '';
    
    try {
        tasks.forEach((task, index) => {
            const card = template.content.cloneNode(true);
            card.querySelector('.task-id .highlight').textContent = task.subID;
            let statusText, statusClass;
            switch(task.status) {
                case 0:
                    statusText = '创建任务';
                    statusClass = 'creating';
                    break;
                case 1:
                    statusText = '任务分发';
                    statusClass = 'distributed';
                    break;
                case 2:
                    statusText = '处理中';
                    statusClass = 'processing';
                    break;
                case 3:
                    statusText = '已完成';
                    statusClass = 'completed';
                    break;
                case 4:
                    statusText = '任务失败';
                    statusClass = 'failed';
                    break;
                default:
                    statusText = '未知状态';
                    statusClass = 'unknown';
            }
            const statusEl = card.querySelector('.task-status');
            statusEl.textContent = statusText;
            statusEl.className = `task-status ${statusClass}`;
            
            // 时间信息
            const timeItems = card.querySelectorAll('.time-item');
            timeItems[0].querySelector('.time-value').textContent = task.publishTime;
            timeItems[1].querySelector('.time-value').textContent = task.startTime;
            timeItems[2].querySelector('.time-value').textContent = task.endTime;
            timeItems[3].querySelector('.time-value').textContent = task.duration;
            
            // 节点信息
            const infoItems = card.querySelectorAll('.info-item');
            infoItems[0].querySelector('.info-value').textContent = task.systemUUID;
            infoItems[1].querySelector('.info-value').textContent = task.nodeIP;
            infoItems[2].querySelector('.info-value').textContent = task.estimatedDuration;
            infoItems[3].querySelector('.info-value').textContent = task.gpuModel;
            
            // 添加卡片动画
            const cardEl = card.querySelector('.task-card');
            cardEl.style.transitionDelay = `${index * 50}ms`;
            
            container.appendChild(card);
        });
        
    } catch (error) {
        errorEl.style.display = 'block';
        console.error("渲染任务卡片出错:", error);
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
        const tasks = await sendMyTasks(userName);
        renderTasks(tasks);
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