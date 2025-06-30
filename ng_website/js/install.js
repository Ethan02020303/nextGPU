
// 页面加载完成后执行
document.addEventListener('DOMContentLoaded', async function() {
    const loader = document.getElementById('loader');
    const errorEl = document.getElementById('error');
    const container = document.getElementById('markdown-container');
    try {
        //获取用户信息
        const {logined, userName} = checkLoginStatus();
        if(logined){
            sendUserInfo(userName)
        }else{
            sendUserInfo()
        }
        
        // 加载Markdown文件
        loadMarkdownFile('files/install.md');

        //加载
        async function loadMarkdownFile(filePath) {
            try {
                // 显示加载中
                loader.style.display = 'block';
                errorEl.style.display = 'none';
                
                // 解决本地文件系统访问问题
                if (window.location.protocol === 'file:') {
                    // 使用XMLHttpRequest代替fetch
                    const markdownText = await loadLocalFile(filePath);
                    renderMarkdown(markdownText);
                } else {
                    // 正常使用fetch
                    const response = await fetch(filePath);
                    if (!response.ok) throw new Error('加载失败');
                    const markdownText = await response.text();
                    renderMarkdown(markdownText);
                }
            } catch (error) {
                console.error('加载Markdown失败:', error);
                loader.style.display = 'none';
                errorEl.style.display = 'block';
                errorEl.textContent = `加载失败: ${error.message}`;
            }
        }

        function renderMarkdown(markdownText) {
            // 将Markdown转换为HTML
            const htmlContent = marked.parse(markdownText);
            container.innerHTML = htmlContent;
            loader.style.display = 'none';
        }

        function loadLocalFile(filePath) {
            return new Promise((resolve, reject) => {
                const xhr = new XMLHttpRequest();
                xhr.open('GET', filePath, true);
                xhr.onreadystatechange = function() {
                    if (xhr.readyState === 4) {
                        if (xhr.status === 0 || xhr.status === 200) {
                            resolve(xhr.responseText);
                        } else {
                            reject(new Error(`加载失败: ${xhr.status}`));
                        }
                    }
                };
                xhr.onerror = function() {
                    reject(new Error('网络错误'));
                };
                xhr.send();
            });
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