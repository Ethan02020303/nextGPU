document.addEventListener('DOMContentLoaded', function() {

    //获取登录信息
    const userName = localStorage.getItem('userName');
    if(userName !== "") {
        sendUserInfo(userName)
    }else{
        sendUserInfo()
    }

    // 标签页切换功能
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            // 移除所有按钮的active类
            tabButtons.forEach(btn => btn.classList.remove('active'));
            // 添加当前按钮的active类
            button.classList.add('active');
            
            // 获取要激活的标签ID
            const tabId = button.getAttribute('data-tab');
            
            // 隐藏所有标签内容
            tabContents.forEach(content => {
                content.classList.remove('active');
            });
            
            // 显示选中的标签内容
            document.getElementById(tabId).classList.add('active');
        });
    });
    
    // 复制按钮功能
    const copyButtons = document.querySelectorAll('.copy-btn');
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            // 获取代码块元素
            const codeBlock = this.closest('.api-demo-block').querySelector('.api-demo-code');
            
            // 创建临时textarea来复制文本
            const tempTextArea = document.createElement('textarea');
            tempTextArea.value = codeBlock.textContent;
            document.body.appendChild(tempTextArea);
            
            // 选中并复制文本
            tempTextArea.select();
            document.execCommand('copy');
            
            // 移除临时textarea
            document.body.removeChild(tempTextArea);
            
            // 更新按钮文本，提示复制成功
            const originalText = this.textContent;
            this.textContent = '✓ 已复制';
            
            // 3秒后恢复原始文本
            setTimeout(() => {
                this.textContent = originalText;
            }, 2000);
        });
    });
    
    // 常见问题折叠功能
    const faqQuestions = document.querySelectorAll('.faq-question');
    faqQuestions.forEach(question => {
        question.addEventListener('click', () => {
            const answer = question.nextElementSibling;
            const isOpen = answer.style.display === 'block';
            
            // 关闭所有答案
            document.querySelectorAll('.faq-answer').forEach(ans => {
                ans.style.display = 'none';
            });
            
            // 重置所有箭头
            document.querySelectorAll('.faq-question i').forEach(icon => {
                icon.className = 'fas fa-chevron-down';
            });
            
            // 切换当前答案
            if (!isOpen) {
                answer.style.display = 'block';
                question.querySelector('i').className = 'fas fa-chevron-up';
            }
        });
    });
    
    
    // // 用户数据模拟
    // document.getElementById('user-credits').textContent = '2500';
    // document.getElementById('user-workflows').textContent = '14';
    // document.getElementById('user-images').textContent = '237';
    // document.getElementById('user-storage').textContent = '2.3GB/5GB';
});