//——————————————————分类渲染————————————————————————————
async function sendCategoryRequest(categoryName) {
    try {
        const requestData = {
            categoryName: categoryName,
        };
        const response = await fetch(`${BASE_URL}/getCategorie`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(requestData)
        });
        const data = await response.json();
        // 检查API返回的codeId
        if (data.codeId === 200) {
            // 返回categories数组
            return data.models;
        } else {
            // 非200状态码时抛出错误
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

// 获取分类数据的函数
async function fetchCategories() {
    try {
        const response = await fetch(`${BASE_URL}/getCategories`);
        if (!response.ok) {
            throw new Error(`HTTP错误! 状态码: ${response.status}`);
        }
        const data = await response.json();
        // 检查API返回的codeId
        if (data.codeId === 200) {
            // 返回categories数组
            return data.categories;
        } else {
            // 非200状态码时抛出错误
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        console.error('获取分类数据失败:', error);
        // 返回空数组并显示错误信息
        return [];
    }
}

// 渲染分类标签
function renderCategories(categories) {
    const container = document.getElementById('model-categories');
    // 清空现有内容
    container.innerHTML = '';
    
    // 创建"全部模型"分类
    const allCategory = document.createElement('div');
    allCategory.className = 'category active';
    allCategory.textContent = '全部风格';
    allCategory.dataset.categoryId = 'all';
    allCategory.dataset.categoryName = 'all'; 
    container.appendChild(allCategory);
    
    // 渲染其他分类
    categories.forEach(category => {
        const categoryElement = document.createElement('div');
        categoryElement.className = 'category';
        categoryElement.textContent = category.name;
        categoryElement.dataset.categoryId = category.id;
        categoryElement.dataset.categoryName = category.name; 
        container.appendChild(categoryElement);
    });
    
    // 添加事件监听器
    container.querySelectorAll('.category').forEach(category => {
        category.addEventListener('click', function() {
            // 移除所有active类
            container.querySelectorAll('.category').forEach(cat => {
                cat.classList.remove('active');
            });
            // 添加当前active类
            this.classList.add('active');
            
            // 向后端API发起请求
            const categoryName = this.dataset.categoryName;
            
            // 根据分类ID过滤模型
            filterModelsByCategory(categoryName);
        });
    });
}

// 根据分类ID过滤模型
async function filterModelsByCategory(categoryName) {
    const modelsContainer = document.getElementById('models-container');
    modelsContainer.innerHTML = `
        <div class="loading">
            <div class="loading-spinner"></div>
            <span>正在加载模型...</span>
        </div>
    `;
    
    try {
        let models;
        if (categoryName === 'all') {
            models = await fetchModels();
        } else {
            models = await sendCategoryRequest(categoryName);
        }
        renderModels(models);
    } catch (error) {
        console.error('按分类过滤模型失败:', error);
        modelsContainer.innerHTML = `
            <div class="error-state">
                <i class="fas fa-exclamation-triangle fa-3x"></i>
                <p>模型加载失败，请稍后再试</p>
            </div>
        `;
    }
}

//——————————————————卡片渲染————————————————————————————

// 获取模型数据的函数
async function fetchModels() {
    try {
        const response = await fetch(`${BASE_URL}/getModels`);
        if (!response.ok) {
            throw new Error(`HTTP错误! 状态码: ${response.status}`);
        }
        const data = await response.json();
        // 检查API返回的codeId
        if (data.codeId === 200) {
            // 返回models数组
            return data.models;
        } else {
            // 非200状态码时抛出错误
            throw new Error(`API错误: ${data.msg || '未知错误'}`);
        }
    } catch (error) {
        console.error('获取模型数据失败:', error);
        // 返回空数组并显示错误信息
        return [];
    }
}

// 添加导航函数
function navigateToDetailPage(model) {
    console.log("click navigateToDetailPage");
    sessionStorage.setItem('selectedWorkflow', JSON.stringify(model));

    const encodeParam = encodeURIComponent(model.workflowTitle)
    window.location.href = `generate.html?title=${encodeParam}`;
}

// 渲染模型卡片
// 渲染模型卡片
function renderModels(models) {
    const container = document.getElementById('models-container');
    const template = document.getElementById('model-card-template');
    // 清空现有内容
    container.innerHTML = '';
    // 遍历模型数据并创建卡片
    models.forEach(model => {
        const clone = document.importNode(template.content, true);
        // 填充图片
        const image = clone.querySelector('.model-image');
        const imageUrl = model.workflowCover || 'default-image.jpg';
        image.style.backgroundImage = `url('${imageUrl}')`;
        // 设置徽章
        const badge = clone.querySelector('.model-badge');
        if (model.badge === 0) {
            badge.textContent = "免费";
            badge.style.backgroundColor = "#f59e0b";
        } else if (model.badge === 1) {
            badge.textContent = "提成";
            badge.style.backgroundColor = "#2563eb";
        }  else {
            badge.style.display = "none";
        }
        // 设置标题
        clone.querySelector('.model-title').textContent = model.workflowTitle;
        // 设置作者信息
        const authorAvatar = clone.querySelector('.author-avatar');
        authorAvatar.textContent = model.authorName.charAt(0);
        // 随机生成头像背景色
        const colors = [
            'linear-gradient(135deg, #f59e0b, #ff9e6d)',
            'linear-gradient(135deg, #3b82f6, #8b5cf6)',
            'linear-gradient(135deg, #10b981, #06b6d4)',
            'linear-gradient(135deg, #ef4444, #f97316)'
        ];
        authorAvatar.style.background = colors[Math.floor(Math.random() * colors.length)];
        const authorName = model.authorName || '匿名用户';
        clone.querySelector('.author-name').textContent = authorName;
        // 设置统计信息
        const stats = clone.querySelector('.model-stats');
        // 格式化下载量
        const formatCount = (count) => {
            if (count >= 10000) return (count / 10000).toFixed(1) + '万';
            if (count >= 1000) return (count / 1000).toFixed(1) + '千';
            return count;
        };
        stats.querySelector('.downloads').textContent = `${formatCount(model.downloadCount)} 使用`;
        stats.querySelector('.likes').textContent = `${formatCount(model.likeCount)} 点赞`;
        // stats.querySelector('.comments').textContent = `${formatCount(model.commentCount)} 评论`;

        //添加点击跳转
        const card = clone.querySelector('.model-card');
        card.addEventListener('click', function() {
            navigateToDetailPage(model);
        });

        container.appendChild(clone);
    });
}


// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', async function() {
    try {
        const userName = localStorage.getItem('userName');
        if(userName !== "") {
            // 获取用户信息
            sendUserInfo(userName)
        }else{
            sendUserInfo()
        }
        // 渲染分类标签
        const categories = await fetchCategories();
        renderCategories(categories);
        
        // 渲染模型卡片
        const models = await fetchModels();
        renderModels(models);
        
        // 如果没有数据，显示空状态
        if (models.length === 0) {
            document.getElementById('models-container').innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-images fa-3x"></i>
                    <p>暂无模型数据</p>
                </div>
            `;
        }
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