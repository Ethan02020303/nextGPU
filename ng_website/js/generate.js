
// 日期格式化函数
function formatDate(dateString) {
    const date = new Date(dateString);
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    const hour = date.getHours();
    const minute = date.getMinutes();
    const second = date.getSeconds()
    return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
}

function getFormattedTimestamp() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0'); // 月份从0开始，所以+1
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    const seconds = String(now.getSeconds()).padStart(2, '0');
    return `${year}${month}${day}${hours}${minutes}${seconds}`;
}

// 将模型数据按日期分组
function groupModelsByDate(models) {
    const grouped = {};
    
    models.forEach(model => {
        if (!grouped[model.createDate]) {
            grouped[model.createDate] = [];
        }
        grouped[model.createDate].push(model);
    });
    
    return grouped;
}

// 添加导航函数
function navigateToDetailPage(model) {
    console.log("click navigateToDetailPage");
    sessionStorage.setItem('selectedWorkflow', JSON.stringify(model));
    window.location.href = 'generate.html';
}

function downloadImage(url, filename) {
    // 创建一个a标签
    const a = document.createElement('a');
    a.href = url;
    a.download = filename || 'download.jpg'; // 设置下载文件名
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
}

async function getParam(name) {
    const queryString = window.location.search;
    if(queryString) {
        const params = new URLSearchParams(queryString);
        const paramValue = params.get(name);
        if(paramValue !== null) {
            const decodedParam = decodeURIComponent(paramValue);
            console.log('工作流: ', decodedParam);
            return decodedParam
        } 
        console.log(`未找到参数: ${name}`);
    } 
    console.log('未找到查询字符串');
    return ""
}

async function getTitle(workflowTitle) {    
    console.log('工作流: ', workflowTitle);
    workflow = await sendTitle(workflowTitle)
    console.log('工作流内容: ', workflow);
    createWorkflow(workflow)
}

// 按日期分组渲染模型
function renderModelsGroupedByDate(tasks) {
    const container = document.getElementById('models-container');
    container.innerHTML = '';
    const groupedModels = {};
    tasks.forEach(task => {
        const taskID = task.taskID;
        const inputData = JSON.parse(task.input);
        const inputParameters = inputData.parameters;
        const ratio = inputParameters.imageRatio;
        const prompt = inputParameters.promptText;
        const imageCount = task.subCount;
        const startTime = task.startTime;
        const completionTime = task.completionTime;
        let inputParam = "数量: " + imageCount + ", 比例: " + ratio + ", 起始时间: " + startTime + ", 结束时间："+ completionTime;
        let promptParam = "创意描述: " + prompt;
        
        // 添加日期标题
        const dateHeader = document.createElement('div');
        dateHeader.className = 'date-header';
        dateHeader.innerHTML = `
            <div class="date-title">
                <i class="fas fa-microchip"></i>
                <div class="date-text">${formatDate(task.startTime)}</div>
            </div>
            <div class="date-subtitle">${inputParam}</div>
            <div class="date-subtitle">${promptParam}</div>
            <div class="date-divider"></div>
        `;
        container.appendChild(dateHeader);
        // 创建卡片容器
        const cardRow = document.createElement('div');
        cardRow.className = 'card-row';
        //遍历每一个子任务
        task.subs.forEach(subTask =>{
             const card = createModelCard(subTask);
            cardRow.appendChild(card);
        });
        container.appendChild(cardRow);
    });
}

// 创建模型卡片
function createModelCard(subTask) {
    const template = document.getElementById('model-card-template');
    const clone = document.importNode(template.content, true);
    const image = clone.querySelector('.model-image');
    const imageUrl = subTask.url || 'default-image.jpg';
    image.style.backgroundImage = `url('${imageUrl}')`;
    const downloadBtn = clone.querySelector('.download-btn');
    downloadBtn.setAttribute('data-image-url', imageUrl);
    return clone;
}

function createWorkflow(workflow) {
    // 更新工作流名称
    const modelNameElement = document.querySelector('.model-name');
    if (modelNameElement && workflow.title) {
        modelNameElement.textContent = workflow.title;
    }
    // 更新工作流描述
    const modelDescElement = document.querySelector('.model-description p');
    if (modelDescElement && workflow.description) {
        modelDescElement.textContent = workflow.description;
    }
    // 更新工作流图片
    const modelImageElement = document.querySelector('.model-image-placeholder');
    if (modelImageElement && workflow.cover) {
        modelImageElement.style.backgroundImage = `url('${workflow.cover}')`;
    }
}

// 处理图片生成完成
function handleImageCompleted(data) {
  const { subID, imageUrl } = data;
  
  const card = document.querySelector(`.model-card[data-sub-id="${subID}"]`);
  if (!card) return;
  
  const modelImage = card.querySelector('.model-image');
  const progressContainer = card.querySelector('.progress-ring');
  
  // 移除进度显示
  if (progressContainer) {
    progressContainer.remove();
  }
  
  // 设置实际图片
  modelImage.style.backgroundImage = `url('${imageUrl}')`;
  modelImage.style.backgroundColor = 'transparent';
  
  // 更新下载按钮
  const downloadBtn = card.querySelector('.download-btn');
  downloadBtn.setAttribute('data-image-url', imageUrl);
}

// 更新子任务等待进度
function updateSubProgress(subTasks) {
    subTasks.forEach(subTask => {
        console.log(subTask)
        //获取子任务信息
        const subID = subTask.subID;
        const state = subTask.status;
        const initWaitCount = subTask.initWaitCount;
        const spareWaitCount = subTask.spareWaitCount;
        const publishTime = subTask.publishTime;
        const startTime = subTask.startTime;
        const estimateMs = subTask.estimateMs;
        const endTime = subTask.endTime;
        const subCard = document.querySelector(`.model-card[data-sub-id="${subID}"]`);
        if(subCard === null){
            return
        }
        
        const progressContainer = subCard.querySelector('.progress-container');
        if(progressContainer) progressContainer.style.display = 'block';

        // 获取文本状态元素
        const queueStatus = subCard.querySelector('.queue-status');
        const progressStatus = subCard.querySelector('.progress-status');
        
        // 重置所有状态的显示
        if(queueStatus) queueStatus.classList.add('hidden');
        if(progressStatus) progressStatus.classList.add('hidden');

        //根据状态显示
        if (state === 0 || state === 1) {
            //等待分发
            if(queueStatus) {
                queueStatus.classList.remove('hidden');
                queueStatus.textContent = `等待中 ${spareWaitCount}/${initWaitCount}`;
            }
        } else if (state === 2) {
            //已经分发
            if(queueStatus) {
                queueStatus.classList.remove('hidden');
                queueStatus.textContent = `队列中 ${spareWaitCount}/${initWaitCount}`;
            }
        } else if (state === 3) {
            //正在执行
            if(progressStatus) {
                progressStatus.classList.remove('hidden');
                
                // 记录开始时间（如果服务器没有提供，使用当前时间）
                const executionStartTime = startTime ? new Date(startTime).getTime() : Date.now();
                
                // 每秒更新进度
                subCard.progressTimer = setInterval(() => {
                    const now = Date.now();
                    const elapsed = now - executionStartTime;
                    let processProgress = Math.min(100, (elapsed / estimateMs) * 100);
                    
                    // 更新显示
                    progressStatus.textContent = `处理中 ${Math.round(processProgress)}%`;
                    
                    // 如果进度完成，清除定时器
                    if (processProgress >= 100) {
                        clearInterval(subCard.progressTimer);
                        delete subCard.progressTimer;
                    }
                }, 1000); // 每秒更新一次
            }
        } else if(state === 4) {
            //生成成功

            // 清除定时器（如果存在）
            if (subCard.progressTimer) {
                clearInterval(subCard.progressTimer);
                delete subCard.progressTimer;
            }

            const progressContainer = subCard.querySelector('.progress-container');
            if(progressContainer) {
                progressContainer.style.display = "none";
                progressContainer.remove();
            }

            //显示生成内容
            const modelImage = subCard.querySelector('.model-image');
            const imageUrl = subTask.ossUrls[0] || 'default-image.jpg';
            modelImage.classList.add('model-image-success');

            modelImage.style.backgroundImage = `url('${imageUrl}')`;
            modelImage.style.backgroundColor = 'transparent';
            modelImage.style.backgroundSize = 'cover';
            modelImage.style.backgroundPosition = 'center';
            modelImage.style.backgroundRepeat = 'no-repeat';

            const downloadBtn = subCard.querySelector('.download-btn');
            downloadBtn.classList.remove('hidden');
            downloadBtn.setAttribute('data-image-url', imageUrl);
        } else if (state === 5) {
            //生成失败
            showGenerationError({
                subID,
                error: "生成失败"
            });
        }
    });
}

window.addEventListener('onReadMessage', function(event) {
  const msg = event.detail.message;
  console.log('收到WebSocket消息:', msg);
  if(msg.messageID === 1) { //同步任务进度
    if(msg.event === "SessionSync" || msg.event === "ImageGenerateSuccess" || msg.event === "InLocalQueue" || msg.event === "GenerateStart"){
        const subTasks = msg.tasks;
        updateSubProgress(subTasks)
    }
  }
});

window.addEventListener('onDisconnectMessage', function(event) {
  const messageData = event.detail.message;
  console.log('收到WebSocket断开消息:', messageData);
});

window.addEventListener('onErrorMessage', function(event) {
  const messageData = event.detail.message;
  console.log('收到WebSocket错误消息:', messageData);
});

document.addEventListener('DOMContentLoaded', async function() {
    //解析传递的参数
    const workflowTitle = await getParam('title') || "";
    if (workflowTitle && typeof workflowTitle === 'string' && workflowTitle.trim() !== "") {
        getTitle(workflowTitle)
    }

    //获取登录信息
    const userName = localStorage.getItem('userName');
    if(userName !== "") {
        // 获取用户信息
        sendUserInfo(userName)
        // 渲染生成的卡片
        const tasks = await sendMyTask(userName);
        renderModelsGroupedByDate(tasks);
    }else{
        sendUserInfo()
    }
    // 比例选择交互
    const options = document.querySelectorAll('.ratio-option');
    options.forEach(option => {
        option.addEventListener('click', function() {
            options.forEach(opt => opt.classList.remove('active'));
            this.classList.add('active');
        });
    });
    // 生成按钮效果
    const generateBtn = document.querySelector('.generate-btn');
    if (generateBtn) {
        generateBtn.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-2px)';
        });
        generateBtn.addEventListener('mouseleave', function() {
            this.style.transform = '';
        });
    }

    // 为返回按钮添加功能
    document.querySelector('.back-btn').addEventListener('click', function() {
        console.log('返回按钮被点击');
        window.location.href = 'market.html';
    });

    // 点击下载按钮
    document.getElementById('models-container').addEventListener('click', function(e) {
        const downloadBtn = e.target.closest('.download-btn');
        if (downloadBtn) {
            const imageUrl = downloadBtn.getAttribute('data-image-url');
            if (imageUrl) {
                downloadImage(imageUrl, 'image.jpg');
            }
        }
    });

    // 点击新生成的下载按钮
    document.getElementById('generating-container').addEventListener('click', function(e) {
        const downloadBtn = e.target.closest('.download-btn');
        if (downloadBtn) {
            const imageUrl = downloadBtn.getAttribute('data-image-url');
            if (imageUrl) {
                downloadImage(imageUrl, 'image.jpg');
            }
        }
    });

    // 图片数量选择器
    const decreaseBtn = document.getElementById('decrease-btn');
    const increaseBtn = document.getElementById('increase-btn');
    const quantityValue = document.getElementById('quantity-value');
    const costValue = document.getElementById('cost-value');
    if (decreaseBtn && increaseBtn && quantityValue && costValue) {
        let quantity = 4; 
        const costPerImage = 5; 
        
        // 更新消耗积分显示
        function updateCost() {
            const totalCost = quantity * costPerImage;
            costValue.textContent = totalCost;
        }
        decreaseBtn.addEventListener('click', function(event) {
            event.preventDefault();
            if (quantity > 1) {
                quantity--;
                quantityValue.textContent = quantity;
                updateCost();
            }
            decreaseBtn.disabled = quantity <= 1;
            increaseBtn.disabled = false;
        });
        increaseBtn.addEventListener('click', function(event) {
            event.preventDefault();
            if (quantity < 20) {
                quantity++;
                quantityValue.textContent = quantity;
                updateCost();
            }
            increaseBtn.disabled = quantity >= 20;
            decreaseBtn.disabled = false;
        });
        decreaseBtn.disabled = quantity <= 1;
        increaseBtn.disabled = quantity >= 20;
        updateCost();
    } else {
        console.error("无法找到数量选择器元素");
    }
    // 风格选择
    const styleItems = document.querySelectorAll('.style-item');
    styleItems.forEach(item => {
        item.addEventListener('click', function() {
            styleItems.forEach(style => style.classList.remove('active'));
            this.classList.add('active');
        });
    });
    // 生成按钮功能
    if (generateBtn) {
        generateBtn.addEventListener('click', async function() {
            //生成数量
            const quantityElement = document.getElementById('quantity-value');
            const imageCount = quantityElement ? parseInt(quantityElement.textContent) : 3; 
            //生成任务ID
            const today = new Date();
            let taskID = "nextGPU"+ getFormattedTimestamp();
            //比例
            const activeRatioOption = document.querySelector('.ratio-option.active');
            const selectedRatio = activeRatioOption.dataset.ratio;
            //描述词
            const promptTextarea = document.querySelector('.creative-content .input-box textarea');
            let promptParam = promptTextarea ? promptTextarea.value : '';
            //工作流title
            const res = await sendNewTask(workflowTitle, imageCount, "2025/05", taskID, selectedRatio, promptParam)
            const subs = res.subs;
            //启动websocket监听每一个子任务的进度
            connect(res.connect)
            
            //设置页面
            const container = document.getElementById('generating-container');
            const newContent = document.createElement('div');
            const dateHeader = document.createElement('div');
            dateHeader.className = 'date-header';

            let curTime = today.getFullYear() +"-" + today.getMonth()+1 + "-" + today.getDate() + " " + today.getHours()+ ":"+ today.getMinutes()+":"+today.getSeconds();
            let inputParam = "数量: " + imageCount + ", 比例: " + selectedRatio + ", 起始时间: " + curTime;            
            promptParam = "创意描述: " + promptParam

            dateHeader.innerHTML = `
                <div class="date-title">
                    <i class="fas fa-microchip"></i>
                    <div class="date-text">${curTime}</div>
                </div>
                <div class="date-subtitle">${inputParam}</div>
                <div class="date-subtitle">${promptParam}</div>
                <div class="date-divider"></div>
            `;
            newContent.appendChild(dateHeader);
            const cardRow = document.createElement('div');
            cardRow.className = 'card-row';

            for (let i = 0; i < imageCount; i++) {
                const sub = subs[i];
                const template = document.getElementById('generated-card-template');
                const clone = document.importNode(template.content, true);
                
                const card = clone.querySelector('.model-card');
                card.setAttribute('data-sub-id', sub.subID);
                
                // 只设置占位背景，不要在此处创建进度环
                const modelImage = clone.querySelector('.model-image');
                modelImage.style.backgroundColor = '#e2e8f0';
                modelImage.style.backgroundImage = `
                    linear-gradient(45deg, rgba(203, 213, 225, 0.3) 25%, transparent 25%), 
                    linear-gradient(-45deg, rgba(203, 213, 225, 0.3) 25%, transparent 25%),
                    linear-gradient(45deg, transparent 75%, rgba(203, 213, 225, 0.3) 75%),
                    linear-gradient(-45deg, transparent 75%, rgba(203, 213, 225, 0.3) 75%)
                `;
                modelImage.style.backgroundSize = '20px 20px';
                modelImage.style.backgroundPosition = '0 0, 0 10px, 10px -10px, -10px 0px';
                
                cardRow.appendChild(clone);
            }
            newContent.appendChild(cardRow);
            
            // 将新内容添加到现有内容的顶部
            container.insertBefore(newContent, container.firstChild);
        });
    }
});
